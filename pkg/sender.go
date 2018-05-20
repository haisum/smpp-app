package pkg

import (
	"fmt"

	"context"

	"bitbucket.org/codefreak/hsmpp/pkg/logger"
	"bitbucket.org/codefreak/hsmpp/pkg/smtext"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

var (
	snd Sender
)

// GetSender returns snd object
func GetSender() Sender {
	return snd
}

// ConnectFiorix connects to pkg server given by addr, user and passwd
// This function triggers a go routine that checks for pkg connection status
// If connection is lost at some point, this retries after 10 seconds.
// Channel fiorix.Connected is filled if pkg gets connected. Other routines
// that depend on pkg connection should wait for Connected channel before
// proceeding.
func ConnectFiorix(ctx context.Context, tx *smpp.Transceiver) error {
	s := &fiorix{
		logger: logger.FromContext(ctx),
	}
	s.tx = tx
	s.conn = s.tx.Bind() // make persistent connection.
	select {
	case c := <-s.conn:
		st := c.Status()
		s.logger.Info("st", st, "msg", "SMPP connection status changed.")
		if st != smpp.Connected {
			return fmt.Errorf("error in establising connection. Status: %s, Error: %s", c.Status(), c.Error())
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	snd = s
	return nil
}

// Sender is implemented by pkg fiorix client code or mock test object
type Sender interface {
	Send(src, dst, enc, msg string, isFlash bool) (string, error)
	SplitLong(src, dst, enc, msg string, isFlash bool) (*smpp.ShortMessage, []pdu.Body)
	SendPart(sm *smpp.ShortMessage, p pdu.Body) (string, error)
	Close() error
	SetFields(p PduFields)
	GetFields() PduFields
	Monitor(status chan<- uint8)
}

// fiorix holds pkg transmitter and a channel indicating when pkg connection
// becomes connected.
type fiorix struct {
	tx     *smpp.Transceiver
	fields PduFields
	conn   <-chan smpp.ConnStatus
	logger logger.Logger
}

// Monitor checks for pkg connection status, if it becomes not connected, it returns error status on channel
// error status is one of last three of following constants:
// Connected fiorix.ConnStatusID = iota + 1
// Disconnected
// ConnectionFailed
// BindFailed
// Caller should listen on provided channel and take appropriate action when error status is returned
func (s *fiorix) Monitor(status chan<- uint8) {
	go func(ch chan<- uint8) {
		s.logger.Info("Listening for connection status change.")
		for c := range s.conn {
			st := c.Status()
			s.logger.Info("st", st, "msg", "SMPP connection status changed.")
			if st != smpp.Connected {
				s.logger.Error("msg", "error in establishing connection", "status", c.Status(), "error", c.Error())
				ch <- uint8(c.Status())
			}
		}
	}(status)
}

// Close closes connection with pkg provider
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
			s.logger.Error(
				"Src", src,
				"Dst", dst,
				"Enc", enc,
				"Text", msg,
				"sm", sm,
				"msg", "Error in processing sms request because pkg is not connected.")
		}
		return "", err
	}
	return sm.RespID(), nil
}

// SplitLong splits a long message in parts and returns pdu.Body which can be sent individually using SendPart method
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
			s.logger.Error(
				"sm", sm,
				"p", p,
				"msg", "error in processing partial sms send request because pkg is not connected")
		}
		return "", err
	}
	return sm.RespID(), nil
}
