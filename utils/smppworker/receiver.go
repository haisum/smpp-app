package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/message"
	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
)

func receiver(p pdu.Body) {
	if p.Header().ID == pdu.DeliverSMID {
		go saveDeliverySM(p)
	} else {
		fields := log.Fields{
			"pdu":    p.Header().ID.String(),
			"fields": p.Fields(),
		}
		log.WithFields(fields).Info("PDU Received.")
	}
}

func saveDeliverySM(pdu pdu.Body) {
	deliverSM := pdu.Fields()
	tlvFields := pdu.TLVFields()
	var id string
	var err error
	if val, ok := tlvFields[pdufield.ReceiptedMessageID]; ok {
		b := val.Bytes()
		n := bytes.Index(b, []byte{0})
		id = string(b[:n])
	} else if val, ok := deliverSM["short_message"]; ok {
		id, err = splitShortMessage(val.String(), "id:")
		if err != nil {
			log.Info("Couldn't find id, executing receiver")
			callReceiver(deliverSM)
			return
		}
	} else {
		log.WithField("deliverySM", deliverSM).Error("Couldn't find short_message field or receipted message id")
		return
	}
	deliveryMap := make(map[string]string, len(deliverSM))
	for k, v := range deliverSM {
		deliveryMap[string(k)] = v.String()
	}
	status, _ := splitShortMessage(deliverSM["short_message"].String(), "stat:")
	if status == "DELIVRD" {
		status = string(message.Delivered)
	} else {
		status = string(message.Delivered)
	}
	message.SaveDelivery(id, status)
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
