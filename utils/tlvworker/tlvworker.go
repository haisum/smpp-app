package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	smpp "github.com/CodeMonkeyKevin/smpp34"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

const (
	//MaxLatinChars is number of characters allowed in single latin encoded text message
	MaxLatinChars int = 140
	//MaxUCSChars is number of characters allowed in single ucs encoded text message
	MaxUCSChars int = 50
	// Latin1Type is hexcode for pdu encoding latin
	Latin1Type int = 0x03
	//UCS2Type iis hexcode for pdu encoding UCS2
	UCS2Type int = 0x08
	// SarMsgRefNum is hexcode for sar_msg_refnum tlv field
	SarMsgRefNum int = 0x020C
	// SarTotalSegments is hexcode for sar_total_segments tlv field
	SarTotalSegments int = 0x020E
	// SarSegmentSeqnum is hexcode for sar_seq_num tlv field
	SarSegmentSeqnum int = 0x020F
)

var (
	host     = flag.String("host", "localhost", "SMPP host address.")
	port     = flag.Int("port", 2775, "SMPP host port.")
	username = flag.String("username", "", "Username to connect to smpp server.")
	password = flag.String("password", "", "Password to connect to smpp server.")
	message  = flag.String("message", "Hello world", "Message to send.")
	dst      = flag.String("dst", "", "Destination number.")
	src      = flag.String("src", "", "Source from which message is sent.")
	isUCS    = flag.Bool("isUCS", false, "Set this flag if data should be sent as UCS instead of latin.")
)

func packUI16(n uint16) (b []byte) {
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return
}

func packUI8(n uint8) (b []byte) {
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b[1:]
}

func main() {
	optionalFields := []string{"source_addr_ton", "source_addr_npi", "dest_addr_ton", "dest_addr_npi"}
	optionalFlags := make(map[string]*int)
	for _, v := range optionalFields {
		optionalFlags[v] = flag.Int(v, 0, fmt.Sprintf("optional %s field", v))
	}

	flag.Parse()
	if *username == "" || *password == "" || *dst == "" || *src == "" {
		flag.Usage()
		os.Exit(1)
	}
	// connect and bind
	trx, err := smpp.NewTransceiver(
		*host,
		*port,
		5,
		smpp.Params{
			"system_type": "CMT",
			"system_id":   *username,
			"password":    *password,
		},
	)
	if err != nil {
		fmt.Println("Connection Err:", err)
		return
	}
	maxLen := MaxLatinChars
	if *isUCS {
		maxLen = MaxUCSChars
	}
	runeLength := len([]rune(*message))
	rand.Seed(time.Now().UnixNano())
	randRefNum := uint16(rand.Intn(math.MaxUint16))
	msgRefNum := packUI16(randRefNum)

	for i := 0; i < runeLength; i += maxLen {
		var text string
		params := smpp.Params{}
		for _, v := range optionalFields {
			if *optionalFlags[v] != 0 {
				params[v] = *optionalFlags[v]
			}
		}
		end := runeLength
		if runeLength > i+maxLen {
			end = i + maxLen
		}
		msgPart := string([]rune(*message)[i:end])
		if *isUCS {
			text = string(pdutext.UCS2(msgPart).Encode())
			params[smpp.DATA_CODING] = UCS2Type
		} else {
			text = string(pdutext.Latin1(msgPart).Encode())
			params[smpp.DATA_CODING] = Latin1Type
		}
		// Send SubmitSm
		p, err := trx.Smpp.SubmitSm(*src, *dst, text, &params)

		// Pdu gen errors
		if err != nil {
			fmt.Println("SubmitSm err:", err)
			return
		}

		fmt.Printf("SarRef: %d, total:  %d, this: %d\n", randRefNum, (runeLength/maxLen)+1, (i/maxLen)+1)

		p.SetTLVField(SarMsgRefNum, 2, msgRefNum)
		p.SetTLVField(SarSegmentSeqnum, 1, packUI8(uint8((i/maxLen)+1)))
		p.SetTLVField(SarTotalSegments, 1, packUI8(uint8((runeLength/maxLen)+1)))

		err = trx.Write(p)
		// Pdu gen errors
		if err != nil {
			fmt.Println("Write err:", err)
			return
		}

		// Should save this to match with message_id
		fmt.Println("seq:", p.GetHeader().Sequence)

	}

	// start reading PDUs
	for {
		pdu, err := trx.Read() // This is blocking
		if err != nil {
			break
		}

		// Transceiver auto handles EnquireLinks
		switch pdu.GetHeader().Id {
		case smpp.SUBMIT_SM_RESP:
			// message_id should match this with seq message
			fmt.Println("MSG ID: ", string(pdu.GetField("message_id").Value().([]byte)))
		case smpp.DELIVER_SM:
			// received Deliver Sm
			// fmt.Printf("Message %s got delivered.", pdu.GetField("message_id").Value())
			fmt.Println("Message got delivered.")
			// Print all fields
			for _, v := range pdu.MandatoryFieldsList() {
				f := pdu.GetField(v)
				fmt.Println(v, ":", f)
			}

			// Respond back to Deliver SM with Deliver SM Resp
			err := trx.DeliverSmResp(pdu.GetHeader().Sequence, smpp.ESME_ROK)

			if err != nil {
				fmt.Println("DeliverSmResp err:", err)
			}
		default:
			fmt.Println("PDU ID:", pdu.GetHeader().Id)
		}
	}

	fmt.Println("ending...")
}
