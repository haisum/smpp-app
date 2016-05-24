package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/streadway/amqp"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	c      *smpp.Config
	s      *smpp.Sender
	sconn  *smpp.Conn
	connid = flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	group  = flag.String("group", "", "Group name of connection.")
	cmutex smpp.CountMutex
)

// Handler is called by rabbitmq library after a queue has been bound/
// deliveries channel gets data when a new job is to be consumed by worker
// This function should wait for done channel before terminating so that all
// pending jobs should be finished and rabbitmq should be notified about disconnect
func handler(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		// multiple threads may race to read/write this, so we need atomic operation
		cmutex.Lock()
		cur := cmutex.Count
		if cur >= sconn.Size {
			log.Info("Waiting one second before proceeding")
			time.Sleep(time.Second * time.Duration(sconn.Time))
			log.Info("Resuming messages")
			cmutex.Count = 0
		}
		cmutex.Unlock()
		go send(d)
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

func receiver(p pdu.Body) {
	if p.Header().ID == pdu.DeliverSMID {
		go saveDeliverySM(p.Fields())
	} else {
		fields := log.Fields{
			"pdu":    p.Header().ID.String(),
			"fields": p.Fields(),
		}
		log.WithFields(fields).Info("PDU Received.")
	}
}

// This is called per job and as a separate go routing
// This function is responsible for acknowledging the job completion to rabbitmq
// This function also increments count by ceil of number of characters divided by number of characters per message.
// When count reaches a certain number defined per connection, worker waits for time t defined in configuration before resuming operations.
func send(d amqp.Delivery) {
	var i queue.Item
	err := i.FromJSON(d.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"body": d.Body,
		}).Error("Failed in parsing json.")
		d.Nack(false, true)
		return
	}
	charLimit := smpp.MaxLatinChars
	if i.Enc == "UCS" {
		charLimit = smpp.MaxUCSChars
	}
	res := float64(float64(len(i.Msg)) / float64(charLimit))
	total := math.Ceil(res)
	cmutex.Lock()
	cmutex.Count = cmutex.Count + int32(total)
	cmutex.Unlock()
	respId, err := s.Send(i.Src, i.Dst, i.Enc, i.Msg)
	if err != nil {
		log.WithFields(log.Fields{
			"Src":    i.Src,
			"Dst":    i.Dst,
			"err":    err,
			"Enc":    i.Enc,
			"Fields": s.Fields,
		}).Error("Couldn't send message.")
		if err != smppstatus.ErrNotConnected {
			d.Reject(false)
		} else {
			d.Nack(false, true)
		}
		go updateMessage(i.MsgId, respId, sconn.ID, err.Error(), int(total), s.Fields)
	} else {
		log.WithFields(log.Fields{
			"Src":    i.Src,
			"Dst":    i.Dst,
			"Enc":    i.Enc,
			"Fields": s.Fields,
		}).Info("Sent message.")
		d.Ack(false)
		go updateMessage(i.MsgId, respId, sconn.ID, "", int(total), s.Fields)
	}
	log.WithField("RespId", respId).Info("response id")
}

func updateMessage(id, respId, con, errMsg string, total int, fields smpp.PduFields) {
	m, err := models.GetMessage(id)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"id":    id,
		}).Error("Couldn't find message with id")
		return
	}
	m.RespId = respId
	m.Connection = con
	m.Error = errMsg
	m.Total = total
	m.Fields = fields
	if errMsg == "" {
		m.SubmittedAt = time.Now().Unix()
		m.Status = models.MsgSent
	} else {
		m.Status = models.MsgError
	}
	err = m.Update()
	if err != nil {
		log.WithError(err).Error("Couldn't update message.")
	}
}

func saveDeliverySM(deliverSM pdufield.Map) {
	var id string
	log.WithFields(log.Fields{"deliverySM": deliverSM}).Info("Received deliverySM")
	if val, ok := deliverSM["short_message"]; ok {
		log.WithField("ucs", string(pdutext.Raw(deliverSM["short_message"].Bytes()).Decode())).Info("Decoded message")
		var err error
		id, err = splitShortMessage(val.String(), "id:")
		if err != nil {
			log.WithError(err).Error("Couldn't find id")
			return
		}
	} else {
		log.WithField("deliverySM", deliverSM).Error("Couldn't find short_message field")
		return
	}
	criteria := models.MessageCriteria{
		RespId: id,
		// note: dst and src are swapped in deliverSM
		Dst: deliverSM["source_addr"].String(),
		Src: deliverSM["destination_addr"].String(),
	}
	ms, err := models.GetMessages(criteria)
	if err != nil || len(ms) == 0 {
		log.WithFields(log.Fields{
			"error":  err,
			"respId": id,
		}).Error("Couldn't find message with id")
		return
	}
	deliveryMap := make(map[string]string, len(deliverSM))
	for k, v := range deliverSM {
		deliveryMap[string(k)] = v.String()
	}
	ms[0].DeliverySM = deliveryMap
	deliverSM["short_message"].String()
	status, _ := splitShortMessage(deliverSM["short_message"].String(), "stat:")
	if status == "DELIVRD" {
		ms[0].DeliveredAt = time.Now().Unix()
		ms[0].Status = models.MsgDelivered
	} else {
		ms[0].Status = models.MsgNotDelivered
	}
	err = ms[0].Update()
	if err != nil {
		log.WithError(err).Error("Error saving deliverySM")
	}
}

func splitShortMessage(sm, sep string) (string, error) {
	var id string
	tokens := strings.Split(sm, sep)
	if len(tokens) < 2 {
		return id, fmt.Errorf("Couldn't find enough tokens")
	}
	id = strings.Fields(tokens[1])[0]
	return id, nil
}

// When SIGTERM or SIGINT is received, this routine will make sure we shutdown our queues and finish in progress jobs
func gracefulShutdown(r *queue.Rabbit) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		log.Print("Sutting down gracefully.")
		r.Close()
		s.Tx.Close()
		os.Exit(0)
	}()
}

// Binds to rabbitmq queue and listens for all numbers starting with supplied prefixes.
// This function calls handler when a connection is succesfully established
func bind() {
	var err error
	sconn = &smpp.Conn{}
	*sconn, err = c.GetConn(*group, *connid)
	log.WithFields(log.Fields{
		"connid":   *connid,
		"username": sconn.URL,
	}).Info("Connection id")
	log.WithFields(log.Fields{
		"URL":    sconn.URL,
		"User":   sconn.User,
		"Passwd": sconn.Passwd,
		"Conn":   sconn,
		"c":      c,
	}).Info("Dialing")
	s = &smpp.Sender{}
	s.Connect(sconn.URL, sconn.User, sconn.Passwd, receiver)
	s.Fields = sconn.Fields
	log.Info("Waiting for smpp connection")
	<-s.Connected
	cmutex = smpp.CountMutex{}
	log.WithField("conn", sconn).Info("Binding")
	if err != nil {
		log.WithField("connid", connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}
	r, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		os.Exit(1)
	}
	log.WithField("Pfxs", sconn.Pfxs).Info("Binding to routing keys")
	err = r.Bind(*group, sconn.Pfxs, handler)
	if err != nil {
		os.Exit(1)
	}
	//Listen for termination signals from OS
	go gracefulShutdown(r)

}

func main() {
	flag.Parse()
	if *connid == "" {
		flag.Usage()
		os.Exit(1)
	}
	var err error
	c = &smpp.Config{}
	*c, err = models.GetConfig()
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}
	bind()

	forever := make(<-chan int)
	<-forever
}
