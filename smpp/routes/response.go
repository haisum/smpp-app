package routes

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
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
