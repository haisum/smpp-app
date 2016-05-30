package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	log "github.com/Sirupsen/logrus"
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/streadway/amqp"
)

var (
	c        *smpp.Config
	s        *smpp.Sender
	sconn    *smpp.Conn
	connid   = flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	group    = flag.String("group", "", "Group name of connection.")
	throttle chan time.Time
)

// Handler is called by rabbitmq library after a queue has been bound/
// deliveries channel gets data when a new job is to be consumed by worker
// This function should wait for done channel before terminating so that all
// pending jobs should be finished and rabbitmq should be notified about disconnect
func handler(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
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
		go send(i)
		<-time.After(time.Duration(float64(int64(time.Second)*int64(sconn.Time)) / (float64(sconn.Size) / float64(i.Total))))
		d.Ack(false)
	}
	time.NewTicker(d)
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

// This is called per job and as a separate go routing
// This function is responsible for acknowledging the job completion to rabbitmq
// This function also increments count by ceil of number of characters divided by number of characters per message.
// When count reaches a certain number defined per connection, worker waits for time t defined in configuration before resuming operations.
func send(i queue.Item) {
	m, err := models.GetMessage(i.MsgId)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"id":  i.MsgId,
		}).Error("Failed in fetching message from db.")
		return
	}
	respId, err := s.Send(m.Src, m.Dst, m.Enc, m.Msg, i.Total)
	sent := time.Now().UTC().Unix()
	if err != nil {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"err":    err,
			"Enc":    m.Enc,
			"Fields": s.Fields,
		}).Error("Couldn't send message.")
		if err == smppstatus.ErrNotConnected {
			log.Error("SMPP not connected. Aborting worker.")
			//exit code 2, because supervisord wont restart this
			os.Exit(2)
		}
		go updateMessage(m, respId, sconn.ID, err.Error(), s.Fields, sent)
	} else {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"Enc":    m.Enc,
			"Fields": s.Fields,
		}).Info("Sent message.")
		go updateMessage(m, respId, sconn.ID, "", s.Fields, sent)
	}
	log.WithField("RespId", respId).Info("response id")
}

func updateMessage(m models.Message, respId, con, errMsg string, fields smpp.PduFields, sent int64) {
	m.RespId = respId
	m.Connection = con
	m.Error = errMsg
	m.Fields = fields
	m.SentAt = sent
	m.Status = models.MsgSent
	if errMsg != "" {
		m.Status = models.MsgError
	}
	err := m.Update()
	if err != nil {
		log.WithError(err).Error("Couldn't update message.")
	}
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

func saveDeliverySM(deliverSM pdufield.Map) {
	log.WithFields(log.Fields{"deliverySM": deliverSM}).Info("Received deliverySM")
	if val, ok := deliverSM["short_message"]; ok {
		log.WithField("ucs", string(pdutext.Raw(deliverSM["short_message"].Bytes()).Decode())).Info("Decoded message")
		_, err := splitShortMessage(val.String(), "id:")
		if err != nil {
			log.Info("Couldn't find id, executing receiver")
			callReceiver(deliverSM)
			return
		}
	} else {
		log.WithField("deliverySM", deliverSM).Error("Couldn't find short_message field")
		return
	}
	log.Info("Skipping deliver msg due to a bug now. Will save it later.")
	return
}

func callReceiver(deliverSM pdufield.Map) {
	if sconn.Receiver != "" {
		log.WithFields(log.Fields{
			"Receiver":      sconn.Receiver,
			"source_addr":   deliverSM[pdufield.SourceAddr].String(),
			"dest_addr":     deliverSM[pdufield.DestinationAddr].String(),
			"short_message": deliverSM[pdufield.ShortMessage].String(),
		}).Info("Executing Receiver")
		err := exec.Command(sconn.Receiver, deliverSM[pdufield.SourceAddr].String(), deliverSM[pdufield.DestinationAddr].String(), deliverSM[pdufield.ShortMessage].String(), *connid, *group).Run()
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err,
			}).Error("Couldn't execute receiver command.")
		}
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
	select {
	case <-s.Connected:
	case <-time.After(time.Duration(time.Second * 5)):
		log.Error("Timed out waiting for smpp connection. Exiting.")
		os.Exit(2)
	}
	log.WithField("conn", sconn).Info("Binding")
	if err != nil {
		log.WithField("connid", connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}
	r, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		os.Exit(2)
	}
	log.WithField("Pfxs", sconn.Pfxs).Info("Binding to routing keys")
	err = r.Bind(*group, sconn.Pfxs, handler)
	if err != nil {
		os.Exit(2)
	}
	//Listen for termination signals from OS
	go gracefulShutdown(r)

}

func main() {
	flag.Parse()
	if *connid == "" {
		flag.Usage()
		os.Exit(2)
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
