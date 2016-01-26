package smpp

import (
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"log"
	"time"
)

type Sender struct {
	Tx *smpp.Transmitter
}

func (s *Sender) Connect(addr, user, passwd string) {
	s.Tx = &smpp.Transmitter{
		Addr:   addr,
		User:   user,
		Passwd: passwd,
	}
	conn := s.Tx.Bind() // make persistent connection.
	go func() {
		for c := range conn {
			st := c.Status()
			log.Println("SMPP connection status: %s", s)
			if st != smpp.Connected {
				log.Printf("SMPP connection failed. Retrying in 10 seconds...")
				<-time.After(time.Second * 10)
				go s.Connect(addr, user, passwd)
				return
			}
		}
	}()
}

func (s *Sender) Send(src, dst, msg string) (string, error) {
	sm, err := s.Tx.Submit(&smpp.ShortMessage{
		Src:      src,
		Dst:      dst,
		Text:     pdutext.Raw(msg),
		Register: smpp.NoDeliveryReceipt,
	})
	if err != nil {
		if err == smpp.ErrNotConnected {
			log.Printf("Error in processing sms request to %s from %s because smpp is not connected", src, dst)
		}
		return "", err
	}
	return sm.RespID(), nil
}
