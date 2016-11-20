package routes

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/schema"
)

const (
	_SOAPResponse string = `<?xml version="1.0" encoding="utf-8"?>
<SOAP-ENV:Envelope SOAP-ENV:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">
   <SOAP-ENV:Body>
  	%s
   </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`
	// UTF8XMLCHAR is character set of xml requests
	UTF8XMLCHAR = "text/xml;charset=UTF-8"
)

type _SOAPEnvelope struct {
	Body _SOAPBody
}

type _SOAPBody struct {
	Response []byte `xml:",innerxml"`
}

// ParseRequest analyzes a http.Request object and puts request data in passed struct
//
// This function works on Content-Type header.
// If Content-Type is set to application/json, request is considered to be raw json and is parsed as json.
// If Content-Type is set to application/xml or text/xml, request is considered to be xml and is parsed as xml.
// If Content-Type is set to text/xml;charset=UTF-8 or application/xml+soap, request is parsed as SOAP.
func ParseRequest(r http.Request, v interface{}) error {
	if cType, ok := r.Header["Content-Type"]; ok && cType[0] == "application/json" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&v)
		if err != nil {
			log.WithError(err).Error("Couldn't understand json request.")
			return err
		}
	} else if cType, ok := r.Header["Content-Type"]; ok && (cType[0] == "application/xml" || cType[0] == "text/xml") {
		decoder := xml.NewDecoder(r.Body)
		err := decoder.Decode(&v)
		if err != nil {
			log.WithError(err).Error("Couldn't understand xml request.")
			return err
		}
	} else if cType, ok := r.Header["Content-Type"]; ok && (cType[0] == UTF8XMLCHAR || cType[0] == "application/xml+soap") {
		decoder := xml.NewDecoder(r.Body)
		var env _SOAPEnvelope
		err := decoder.Decode(&env)
		if err != nil {
			log.WithError(err).Error("Couldn't decode SOAP request. Is it a valid SOAP?")
			return err
		}
		err = xml.Unmarshal(env.Body.Response, &v)
		if err != nil {
			log.WithError(err).Errorf("Couldn't decode SOAP request body in struct. %s", env.Body.Response)
			return err
		}
	} else {
		err := r.ParseForm()
		if err != nil {
			log.WithError(err).Error("Couldn't parse http request form.")
			return err
		}
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)
		err = decoder.Decode(v, r.Form)
		if err != nil {
			log.WithError(err).Error("Couldn't decode form in struct.")
		}
		return err
	}
	return nil
}

// MakeResponse encodes a struct in []byte according to content-type in request object
// json is returned for requests by default
// xml is returned if Content-Type is text/xml or application/xml
// SOAP envelope is returned if Content-Type is text/xml;charset=UTF-8 or application/xml+soap
func MakeResponse(r http.Request, v interface{}) ([]byte, string, error) {
	if cType, ok := r.Header["Content-Type"]; ok && (cType[0] == "application/xml" || cType[0] == "text/xml") {
		b, err := xml.Marshal(v)
		if err != nil {
			log.WithError(err).Error("Couldn't make xml response.")
		}
		return b, UTF8XMLCHAR, err
	} else if cType, ok := r.Header["Content-Type"]; ok && (cType[0] == UTF8XMLCHAR || cType[0] == "application/xml+soap") {
		b, err := xml.Marshal(v)
		if err != nil {
			log.WithError(err).Errorf("Couldn't make SOAP envelope.")
			return b, UTF8XMLCHAR, err
		}
		b = []byte(fmt.Sprintf(_SOAPResponse, b))
		return b, UTF8XMLCHAR, nil
	} else {
		b, err := json.Marshal(v)
		if err != nil {
			log.WithError(err).Error("Couldn't make json response.")
		}
		return b, "application/json", err
	}
}
