package routes

import (
	"encoding/xml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

// Response represents json/xml response we give to requests
type Response struct {
	Obj     interface{} `xml:"Obj" json:"Response"`
	Errors  ResponseErrors
	Ok      bool
	Request interface{}
}

// ResponseErrors is map of errors
type ResponseErrors map[string]string

// ResponseErrors marshals into XML.
func (r ResponseErrors) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	tokens := []xml.Token{start}

	for key, value := range r {
		t := xml.StartElement{Name: xml.Name{"", key}}
		tokens = append(tokens, t, xml.CharData(value), xml.EndElement{t.Name})
	}

	tokens = append(tokens, xml.EndElement{start.Name})

	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	// flush to ensure tokens are written
	err := e.Flush()
	if err != nil {
		return err
	}

	return nil
}

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
