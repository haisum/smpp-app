package smpp

import (
	"encoding/binary"
	smpp "github.com/CodeMonkeyKevin/smpp34"
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"math"
	"math/rand"
	"os"
	"time"
)

const (
	//MaxLatinChars is number of characters allowed in single latin encoded text message
	MaxLatinChars int = 160
	//MaxUCSChars is number of characters allowed in single ucs encoded text message
	MaxUCSChars int = 70

	Latin1Type int = 0x03
	UCS2Type   int = 0x08

	SarMsgRefNum     int = 0x020C
	SarTotalSegments int = 0x020E
	SarSegmentSeqnum int = 0x020F
)

// Sender holds smpp transmitter and a channel indicating when smpp connection
// becomes connected.
type Sender struct {
	Trx *smpp.Transceiver
}

// Connect connects to smpp server given by addr, user and passwd
// This function triggers a go routine that checks for smpp connection status
// If connection is lost at some point, this retries after 10 seconds.
// Channel Sender.Connected is filled if smpp gets connected. Other routines
// that depend on smpp connection should wait for Connected channel before
// proceeding.
func (s *Sender) Connect(host string, port int, username, password string) {
	var err error
	// connect and bind
	s.Trx, err = smpp.NewTransceiver(
		host,
		port,
		5,
		smpp.Params{
			"system_type": "CMT",
			"system_id":   username,
			"password":    password,
		},
	)
	if err != nil {
		log.WithField("err", err).Error("Connection Err")
		os.Exit(1)
	}
}

func packUi16(n uint16) (b []byte) {
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return
}

func packUi8(n uint8) (b []byte) {
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b[1:]
}

//Closes connection to smpp
func (s *Sender) Close() {
	s.Trx.Close()
}

// Send sends sms to given source and destination with latin as encoding
// or ucs if asked.
func (s *Sender) Send(src, dst, message string, isUCS bool, params smpp.Params) error {
	maxLen := MaxLatinChars
	if isUCS {
		maxLen = MaxUCSChars
	}
	runeLength := len([]rune(message))
	var msgRefNum []byte
	total := (runeLength / maxLen) + 1
	if total > 1 {
		rand.Seed(time.Now().UnixNano())
		randRefNum := uint16(rand.Intn(math.MaxUint16))
		msgRefNum = packUi16(randRefNum)
	}

	for i := 0; i < runeLength; i += maxLen {
		var text string
		end := runeLength
		if runeLength > i+maxLen {
			end = i + maxLen
		}
		msgPart := string([]rune(message)[i:end])
		if isUCS {
			text = string(pdutext.UCS2(msgPart).Encode())
			params[smpp.DATA_CODING] = UCS2Type
		} else {
			text = string(pdutext.Latin1(msgPart).Encode())
			params[smpp.DATA_CODING] = Latin1Type
		}
		log.WithFields(log.Fields{
			"src":    src,
			"dst":    dst,
			"text":   text,
			"params": params,
		}).Info("Sending message")
		// Send SubmitSm
		p, err := s.Trx.Smpp.SubmitSm(src, dst, text, &params)

		// Pdu gen errors
		if err != nil {
			log.WithField("err", err).Error("SubmitSm err")
			return err
		}
		if total > 1 {
			p.SetTLVField(SarMsgRefNum, 2, msgRefNum)
			p.SetTLVField(SarSegmentSeqnum, 1, packUi8(uint8((i/maxLen)+1)))
			p.SetTLVField(SarTotalSegments, 1, packUi8(uint8((runeLength/maxLen)+1)))
		}

		err = s.Trx.Write(p)
		// Pdu gen errors
		if err != nil {
			log.WithField("err", err).Error("Write err")
			return err
		}

		// Should save this to match with message_id
		log.WithField("id", p.GetHeader().Sequence).Info("Sent message.")
	}
	return nil
}

func (s *Sender) ReadPDUs() {

	// start reading PDUs
	for {
		pdu, err := s.Trx.Read() // This is blocking
		if err != nil {
			log.WithField("err", err).Error("Error occured.")
			os.Exit(1)
		}

		// Transceiver auto handles EnquireLinks
		switch pdu.GetHeader().Id {
		case smpp.SUBMIT_SM_RESP:
			seq := string(pdu.GetField("message_id").Value().([]byte))
			// message_id should match this with seq message
			log.WithField("MsgId", seq).Info("Acknowledged from smpp.")
		case smpp.DELIVER_SM:
			// received Deliver Sm
			// fmt.Printf("Message %s got delivered.", pdu.GetField("message_id").Value())
			log.Info("Message got delivered.")
			fields := make(log.Fields)
			// Print all fields
			for _, v := range pdu.MandatoryFieldsList() {
				fields[v] = pdu.GetField(v)
			}
			log.WithFields(fields).Info("Got Pdu Fields.")

			// Respond back to Deliver SM with Deliver SM Resp
			err := s.Trx.DeliverSmResp(pdu.GetHeader().Sequence, smpp.ESME_ROK)

			if err != nil {
				log.WithField("err", err).Error("DeliverSmResp err")
			}
		default:
			log.Info("PDU received.")
			fields := make(log.Fields)
			// Print all fields
			for _, v := range pdu.MandatoryFieldsList() {
				fields[v] = pdu.GetField(v)
			}
			log.WithFields(fields).Info("Got Pdu Fields.")
		}
	}
}
