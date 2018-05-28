package response

import (
	"context"
	"encoding/json"
	"net/http"

	"io"

	"github.com/haisum/smpp-app/pkg/logger"
	"github.com/pkg/errors"
)

// Response represents a http response
type Response struct {
	Request interface{}
	Ok      bool
}

// Success represents json/xml response we give to requests
type Success struct {
	Obj interface{} `xml:"Obj" json:"Response"`
	Response
}

// Attachment is returned when we want to return a file for user to download
// One of ReadCloser and Write must be not nil. Otherwise error will be thrown
type Attachment struct {
	Write       func(io.Writer) error
	ReadCloser  io.ReadCloser
	Filename    string
	ContentType string
}

type encoder struct {
	log                logger.Logger
	errFunc            func(err error)
	errResponseHandler func(err error) (errorResponse error, errorCode int)
}

// NewEncoder returns new encoder which can log and call a subscriber errFunc whenever error happens in services
func NewEncoder(log logger.Logger, errFunc func(err error), errResponseHandler func(err error) (errorResponse error, errorCode int)) *encoder {
	return &encoder{log, errFunc, errResponseHandler}
}

// EncodeSuccess encodes a success response from services and sets appropriate headers
func (r *encoder) EncodeSuccess(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	switch response.(type) {
	case Success:
		resp := response.(Success)
		resp.Ok = true
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		return json.NewEncoder(w).Encode(resp)
	case Attachment:
		resp := response.(Attachment)
		w.Header().Set("Content-Type", resp.ContentType)
		w.Header().Set("Content-Disposition", "attachment;filename="+resp.Filename)
		if resp.Write != nil {
			return resp.Write(w)
		} else if resp.ReadCloser != nil {
			_, err := io.Copy(w, resp.ReadCloser)
			defer resp.ReadCloser.Close()
			return err
		}
	}
	return errors.New("couldn't understand given success response")
}

// EncodeError encodes an error response from services and sets appropriate headers
func (r *encoder) EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	errorResponse, errorCode := r.errResponseHandler(err)
	if errorCode == http.StatusForbidden {
		w.Header().Set("WWW-Authenticate", "Basic")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(errorCode)
	encErr := json.NewEncoder(w).Encode(errorResponse)
	if encErr != nil {
		err = errors.Wrap(err, encErr.Error())
	}
	// pass error to handler
	r.errFunc(err)
}
