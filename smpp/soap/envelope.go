package soap

const (
	SOAPResponse string = `<?xml version="1.0" encoding="utf-8"?>
<SOAP-ENV:Envelope SOAP-ENV:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">
   <SOAP-ENV:Body>
  	<ns1:SendSMS2Response xmlns:ns1="smgs">
     	<SendSMS2Result xsi:type="xsd:string">%s</SendSMS2Result>
  	</ns1:SendSMS2Response>
   </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`
)
