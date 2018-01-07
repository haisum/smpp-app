package routes

import (
	"fmt"
	"net/http"

	"encoding/json"
	"encoding/xml"

	log "github.com/Sirupsen/logrus"
)

// Response represents json/xml response we give to requests
type Response struct {
	Obj     interface{} `xml:"Obj" json:"Response"`
	Errors  []ResponseError
	Ok      bool
	Request interface{}
}

// ResponseError is a single error
type ResponseError struct {
	Message string
	Type    string
	Field   string
}

// Error types represent possible values for ResponseError.Type field
const (
	ErrorTypeForm    string = "form"
	ErrorTypeDB      string = "db"
	ErrorTypeAuth    string = "auth"
	ErrorTypeQueue   string = "queue"
	ErrorTypeRequest string = "request"
	ErrorTypeConfig  string = "config"
)

// Send sends a given response with status code
func (resp Response) Send(w http.ResponseWriter, r http.Request, code int) {
	b, cType, err := MakeResponse(r, resp)
	if err != nil {
		log.WithError(err).Error("Couldn't make response.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", cType)
	if code != http.StatusOK {
		w.WriteHeader(code)
	}
	fmt.Fprint(w, string(b))
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
