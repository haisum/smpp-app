package smpp

// PduFields are fields that may be sent to smpp server
// when sending an sms. These are usually optional but some smpp providers
// require them.
type PduFields struct {
	ServiceType          string
	SourceAddrTON        uint8
	SourceAddrNPI        uint8
	DestAddrTON          uint8
	DestAddrNPI          uint8
	ProtocolID           uint8
	PriorityFlag         uint8
	ScheduleDeliveryTime string
	ReplaceIfPresentFlag uint8
	SMDefaultMsgID       uint8
}
