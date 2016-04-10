package smpp

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

const (
	esmClassUdhiMask uint8 = 0x40
	//MaxLatinChars is number of characters allowed in single latin encoded text message
	MaxLatinChars int = 160
	//MaxUCSChars is number of characters allowed in single ucs encoded text message
	MaxUCSChars int = 70
)

// Sender holds smpp transmitter and a channel indicating when smpp connection
// becomes connected.
type Sender struct {
	Tx        *smpp.Transmitter
	Connected chan bool
	Fields    PduFields
}

// Connect connects to smpp server given by addr, user and passwd
// This function triggers a go routine that checks for smpp connection status
// If connection is lost at some point, this retries after 10 seconds.
// Channel Sender.Connected is filled if smpp gets connected. Other routines
// that depend on smpp connection should wait for Connected channel before
// proceeding.
func (s *Sender) Connect(addr, user, passwd string) {
	s.Tx = &smpp.Transmitter{
		Addr:   addr,
		User:   user,
		Passwd: passwd,
	}
	log.WithFields(log.Fields{
		"Addr":   addr,
		"User":   user,
		"Passwd": passwd,
	}).Info("Connected with these credentials.")
	conn := s.Tx.Bind() // make persistent connection.
	s.Connected = make(chan bool, 1)
	go func() {
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
			s.Connected <- true
		}
	}()
}

// Send sends sms to given source and destination with latin as encoding
// or ucs if asked.
func (s *Sender) Send(src, dst, enc, msg string) (string, error) {
	var text pdutext.Codec
	if enc == "ucs" {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Raw(msg)
	}
	maxLen := 134 // 140-6 (UDH)
	rawMsg := text.Encode()
	total := int(len(rawMsg)/maxLen) + 1
	submitFunc := s.Tx.Submit
	if total > 1 {
		submitFunc = s.Tx.SubmitLongMsg
	}
	sm, err := submitFunc(&smpp.ShortMessage{
		Src:                  src,
		Dst:                  dst,
		Text:                 text,
		ServiceType:          s.Fields.ServiceType,
		SourceAddrTON:        s.Fields.SourceAddrTON,
		SourceAddrNPI:        s.Fields.SourceAddrNPI,
		DestAddrTON:          s.Fields.DestAddrTON,
		DestAddrNPI:          s.Fields.DestAddrNPI,
		ProtocolID:           s.Fields.ProtocolID,
		PriorityFlag:         s.Fields.PriorityFlag,
		ScheduleDeliveryTime: s.Fields.ScheduleDeliveryTime,
		ReplaceIfPresentFlag: s.Fields.ReplaceIfPresentFlag,
		SMDefaultMsgID:       s.Fields.SMDefaultMsgID,
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
