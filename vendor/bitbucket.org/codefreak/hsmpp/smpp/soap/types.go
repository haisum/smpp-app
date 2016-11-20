package soap

// Envelope is beginning of soap response
type Envelope struct {
	Body Body
}

// Body is internal contents
type Body struct {
	Request Req `xml:"SendSMS2"`
}

// Req has all input fields
type Req struct {
	Dst        string `xml:"toMobile"`
	Src        string `xml:"sender"`
	Message    string `xml:"smsText"`
	Priority   int    `xml:"priority"`
	Username   string `xml:"username"`
	Password   string `xml:"password"`
	SendAfter  string `xml:"sendAfter"`
	SendBefore string `xml:"sendBefore"`
}
