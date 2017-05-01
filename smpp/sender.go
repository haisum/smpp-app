package smpp

import (
	"bitbucket.org/codefreak/hsmpp/smpp/smtext"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"os"
	"time"
)

var (
	snd Sender
)

// GetSender returns snd object
func GetSender() Sender {
	return snd
}

// Connect connects to smpp server given by addr, user and passwd
// This function triggers a go routine that checks for smpp connection status
// If connection is lost at some point, this retries after 10 seconds.
// Channel fiorix.Connected is filled if smpp gets connected. Other routines
// that depend on smpp connection should wait for Connected channel before
// proceeding.
func ConnectFiorix(tx *smpp.Transceiver) error {
	s := &fiorix{}
	s.tx = tx
	s.conn = s.tx.Bind() // make persistent connection.
	select {
	case c := <-s.conn:
		st := c.Status()
		log.WithField("st", st).Info("SMPP connection status changed.")
		if st != smpp.Connected {
			return fmt.Errorf("Error in establising connection. Status: %s, Error: %s", c.Status(), c.Error())
		}
		return nil
	case <-time.After(time.Second * 5):
		return fmt.Errorf("Timed out waiting for smpp connection.")
	}
	snd = s
	return nil
}

// Sender is implemented by smpp fiorix client code or mock test object
type Sender interface {
	Send(src, dst, enc, msg string, isFlash bool) (string, error)
	SplitLong(src, dst, enc, msg string, isFlash bool) (*smpp.ShortMessage, []pdu.Body)
	SendPart(sm *smpp.ShortMessage, p pdu.Body) (string, error)
	Close() error
	SetFields(p PduFields)
	GetFields() PduFields
	ConnectOrDie()
}

// fiorix holds smpp transmitter and a channel indicating when smpp connection
// becomes connected.
type fiorix struct {
	tx     *smpp.Transceiver
	fields PduFields
	conn   <-chan smpp.ConnStatus
}

// ConnectOrDie checks for smpp connection status, if it becomes not connected, it aborts current application
// This is a blocking function and must be called after a "go" statement in a separate routine
func (s *fiorix) ConnectOrDie() {
	log.Info("Listening for connection status change.")
	for c := range s.conn {
		st := c.Status()
		log.WithField("st", st).Info("SMPP connection status changed.")
		if st != smpp.Connected {
			log.Errorf("Error in establising connection. Status: %s, Error: %s", c.Status(), c.Error())
			os.Exit(2)
		}
	}
}

// Close closes connection with smpp provider
func (s *fiorix) Close() error {
	return s.tx.Close()
}

// SetFields sets pdu fields to given value
func (s *fiorix) SetFields(fields PduFields) {
	s.fields = fields
}

// GetFields gets current pdu fields
func (s *fiorix) GetFields() PduFields {
	return s.fields
}

// Send sends sms to given source and destination with latin as encoding
// or ucs if asked.
func (s *fiorix) Send(src, dst, enc, msg string, isFlash bool) (string, error) {
	var text pdutext.Codec
	if enc == smtext.EncUCS {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Raw(msg)
	}
	sm, err := s.tx.Submit(&smpp.ShortMessage{
		Src:                  src,
		Dst:                  dst,
		Text:                 text,
		ServiceType:          s.fields.ServiceType,
		SourceAddrTON:        s.fields.SourceAddrTON,
		SourceAddrNPI:        s.fields.SourceAddrNPI,
		DestAddrTON:          s.fields.DestAddrTON,
		DestAddrNPI:          s.fields.DestAddrNPI,
		ProtocolID:           s.fields.ProtocolID,
		PriorityFlag:         s.fields.PriorityFlag,
		ScheduleDeliveryTime: s.fields.ScheduleDeliveryTime,
		ReplaceIfPresentFlag: s.fields.ReplaceIfPresentFlag,
		SMDefaultMsgID:       s.fields.SMDefaultMsgID,
		Register:             smpp.FinalDeliveryReceipt,
		IsFlash:              isFlash,
	})
	if err != nil {
		if err == smpp.ErrNotConnected {
			log.WithFields(log.Fields{
				"Src":  src,
				"Dst":  dst,
				"Enc":  enc,
				"Text": msg,
				"sm":   sm,
			}).Error("Error in processing sms request because smpp is not connected.")
		}
		return "", err
	}
	return sm.RespID(), nil
}

//SplitLong splits a long message in parts and returns pdu.Body which can be sent individually using SendPart method
func (s *fiorix) SplitLong(src, dst, enc, msg string, isFlash bool) (*smpp.ShortMessage, []pdu.Body) {
	var text pdutext.Codec
	if enc == smtext.EncUCS {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Raw(msg)
	}
	sm := &smpp.ShortMessage{
		Src:                  src,
		Dst:                  dst,
		Text:                 text,
		ServiceType:          s.fields.ServiceType,
		SourceAddrTON:        s.fields.SourceAddrTON,
		SourceAddrNPI:        s.fields.SourceAddrNPI,
		DestAddrTON:          s.fields.DestAddrTON,
		DestAddrNPI:          s.fields.DestAddrNPI,
		ProtocolID:           s.fields.ProtocolID,
		PriorityFlag:         s.fields.PriorityFlag,
		ScheduleDeliveryTime: s.fields.ScheduleDeliveryTime,
		ReplaceIfPresentFlag: s.fields.ReplaceIfPresentFlag,
		SMDefaultMsgID:       s.fields.SMDefaultMsgID,
		Register:             smpp.FinalDeliveryReceipt,
		IsFlash:              isFlash,
	}
	return sm, s.tx.SplitLong(sm)
}

// SendPart sends a part of long sms obtained from calling SplitLong message
func (s *fiorix) SendPart(sm *smpp.ShortMessage, p pdu.Body) (string, error) {
	var err error
	sm, err = s.tx.SubmitPart(sm, p)
	if err != nil {
		if err == smpp.ErrNotConnected {
			log.WithFields(log.Fields{
				"sm": sm,
				"p":  p,
			}).Error("Error in processing partial sms send request because smpp is not connected.")
		}
		return "", err
	}
	return sm.RespID(), nil
}
