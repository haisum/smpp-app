package routes

import (
	"context"
	"encoding/json"
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"github.com/pkg/errors"
)

type Response struct {
	Request interface{}
	Ok      bool
}

// SuccessResponse represents json/xml response we give to requests
type SuccessResponse struct {
	Obj interface{} `xml:"Obj" json:"Response"`
	Response
}

type responseEncoder struct {
	log                logger.Logger
	errFunc            func(err error)
	errResponseHandler func(err error) (errorResponse error, errorCode int)
}

// NewResponseEncoder returns new encoder which can log and call a subscriber errFunc whenever error happens in services
func NewResponseEncoder(log logger.Logger, errFunc func(err error), errResponseHandler func(err error) (errorResponse error, errorCode int)) *responseEncoder {
	return &responseEncoder{log, errFunc, errResponseHandler}
}

func (r *responseEncoder) EncodeSuccess(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(SuccessResponse)
	resp.Ok = true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}

func (r *responseEncoder) EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
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
