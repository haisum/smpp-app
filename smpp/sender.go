package smpp

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

const (
	esmClassUdhiMask uint8 = 0x40
	//MaxLatinChars is number of characters allowed in single latin encoded text message
	MaxLatinChars = 160
	//MaxUCSChars is number of characters allowed in single ucs encoded text message
	MaxUCSChars = 70
	//EncUCS is string representation of ucs encoding
	EncUCS = "ucs"
	//EncLatin is string representation of latin encoding
	EncLatin = "latin"
)

var (
	snd Sender
)

// SetQueue sets queue equal to object that implements MQ interface. This function shouldn't be used unless you're testing.
// GetQueue takes care of setting a rabbitmq object if q is not set yet.
func setSender(sender Sender) {
	snd = sender
}

// GetSender builds a new sender object that implements Sender interface if s is not already assigned and returns it
func GetSender() Sender {
	if snd == nil {
		setSender(&sender{})
	}
	return snd
}

// Sender is implemented by smpp sender client code or mock test object
type Sender interface {
	Connect(tx SenderTX)
	Send(src, dst, enc, msg string) (string, error)
	SplitLong(src, dst, enc, msg string) (*smpp.ShortMessage, []pdu.Body)
	SendPart(sm *smpp.ShortMessage, p pdu.Body) (string, error)
	Close() error
	SetFields(p PduFields)
	GetFields() PduFields
	ConnectOrDie(conn <-chan smpp.ConnStatus)
}

// SenderTX is implemented by tx object of smpp sender to handle transaction with smpp provider
type SenderTX interface {
	Bind() <-chan smpp.ConnStatus
	Submit(sm *smpp.ShortMessage) (*smpp.ShortMessage, error)
	SplitLong(sm *smpp.ShortMessage) []pdu.Body
	SubmitPart(sm *smpp.ShortMessage, p pdu.Body) (*smpp.ShortMessage, error)
	Close() error
}

// sender holds smpp transmitter and a channel indicating when smpp connection
// becomes connected.
type sender struct {
	tx     SenderTX
	fields PduFields
}

// Connect connects to smpp server given by addr, user and passwd
// This function triggers a go routine that checks for smpp connection status
// If connection is lost at some point, this retries after 10 seconds.
// Channel sender.Connected is filled if smpp gets connected. Other routines
// that depend on smpp connection should wait for Connected channel before
// proceeding.
func (s *sender) Connect(tx SenderTX) {
	s.tx = tx
	conn := s.tx.Bind() // make persistent connection.
	go s.ConnectOrDie(conn)
}

func (s *sender) ConnectOrDie(conn <-chan smpp.ConnStatus) {
	for c := range conn {
		st := c.Status()
		log.WithField("st", st).Info("SMPP connection status changed.")
		if st != smpp.Connected {
			log.WithFields(log.Fields{
				"st":  st,
				"err": c.Error(),
			}).Fatal("SMPP connection failed. Aborting.")
			return
		}
	}
}

// Close closes connection with smpp provider
func (s *sender) Close() error {
	return s.tx.Close()
}

// SetFields sets pdu fields to given value
func (s *sender) SetFields(fields PduFields) {
	s.fields = fields
}

// GetFields gets current pdu fields
func (s *sender) GetFields() PduFields {
	return s.fields
}

// Total counts number of messages in one text string
func Total(msg, enc string) int {
	var text pdutext.Codec
	if enc == EncUCS {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Raw(msg)
	}
	maxLen := 134 // 140-6 (UDH)
	rawMsg := text.Encode()
	return int(len(rawMsg)/maxLen) + 1
}

// Send sends sms to given source and destination with latin as encoding
// or ucs if asked.
func (s *sender) Send(src, dst, enc, msg string) (string, error) {
	var text pdutext.Codec
	if enc == EncUCS {
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
func (s *sender) SplitLong(src, dst, enc, msg string) (*smpp.ShortMessage, []pdu.Body) {
	var text pdutext.Codec
	if enc == EncUCS {
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
	}
	return sm, s.tx.SplitLong(sm)
}

// SendPart sends a part of long sms obtained from calling SplitLong message
func (s *sender) SendPart(sm *smpp.ShortMessage, p pdu.Body) (string, error) {
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
