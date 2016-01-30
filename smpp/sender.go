package smpp

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"time"
)

type Sender struct {
	Tx        *smpp.Transmitter
	Connected chan bool
}

func (s *Sender) Connect(addr, user, passwd string) {
	s.Tx = &smpp.Transmitter{
		Addr:   addr,
		User:   user,
		Passwd: passwd,
	}
	conn := s.Tx.Bind() // make persistent connection.
	s.Connected = make(chan bool, 1)
	go func() {
		for c := range conn {
			st := c.Status()
			log.WithField("st", st).Info("SMPP connection status changed.")
			if st != smpp.Connected {
				log.Error("SMPP connection failed. Retrying in 10 seconds...")
				<-time.After(time.Second * 10)
				go s.Connect(addr, user, passwd)
				return
			} else {
				s.Connected <- true
			}
		}
	}()
}

func (s *Sender) Send(src, dst, enc, msg string) (string, error) {
	var text pdutext.Codec
	if enc == "ucs" {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Latin1(msg)
	}
	sm, err := s.Tx.Submit(&smpp.ShortMessage{
		Src:      src,
		Dst:      dst,
		Text:     text,
		Register: smpp.NoDeliveryReceipt,
	})
	if err != nil {
		if err == smpp.ErrNotConnected {
			log.WithFields(log.Fields{
				"Src":  src,
				"Dst":  dst,
				"Enc":  enc,
				"Text": msg,
			}).Error("Error in processing sms request because smpp is not connected.")
		}
		return "", err
	}
	return sm.RespID(), nil
}
