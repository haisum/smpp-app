package main

import (
	"fmt"
	"os/exec"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"

	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

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
	var id string
	var err error
	if val, ok := deliverSM["short_message"]; ok {
		log.WithField("ucs", string(pdutext.Raw(deliverSM["short_message"].Bytes()).Decode())).Info("Decoded message")
		id, err = splitShortMessage(val.String(), "id:")
		if err != nil {
			log.Info("Couldn't find id, executing receiver")
			callReceiver(deliverSM)
			return
		}
	} else {
		log.WithField("deliverySM", deliverSM).Error("Couldn't find short_message field")
		return
	}
	deliveryMap := make(map[string]string, len(deliverSM))
	for k, v := range deliverSM {
		deliveryMap[string(k)] = v.String()
	}
	status, _ := splitShortMessage(deliverSM["short_message"].String(), "stat:")
	if status == "DELIVRD" {
		status = string(models.MsgDelivered)
	} else {
		status = string(models.MsgNotDelivered)
	}
	models.SaveDelivery(id, deliverSM["destination_addr"].String(), status)
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
