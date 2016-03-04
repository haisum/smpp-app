package soap

type SOAPEnvelope struct {
	Body SOAPBody
}

type SOAPBody struct {
	Request SOAPReq `xml:"SendSMS2"`
}

type SOAPReq struct {
	Dst     string `xml:"toMobile"`
	Src     string `xml:"sender"`
	Message string `xml:"smsText"`
	Coding  int    `xml:"coding"`
}
