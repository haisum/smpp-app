package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/middleware"
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

// ErrorResponse is sent when Error happens in request
type ErrorResponse struct {
	Errors []ResponseError
	Cause  error `xml:"-" json:"-"`
	Response
}

// BadRequestError is sent when user sends invalid request
type BadRequestError error

// AuthErrorResponse is sent when authentication error has happened
type AuthErrorResponse struct {
	ErrorResponse
}

// ForbiddenErrorResponse is sent when access to a resource is forbidden
// usually because of permissions
type ForbiddenErrorResponse struct {
	ErrorResponse
}

type responseEncoder struct {
	log     logger.Logger
	errFunc func(err error)
}

// NewResponseEncoder returns new encoder which can log and call a subscriber errFunc whenever error happens in services
func NewResponseEncoder(log logger.Logger, errFunc func(err error)) *responseEncoder {
	return &responseEncoder{log, errFunc}
}

func (r *responseEncoder) EncodeSuccess(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(SuccessResponse)
	resp.Ok = true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}

func (r *responseEncoder) EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	var (
		errorResponse error
		errorCode     int
	)
	err = errors.Cause(err)
	switch err.(type) {
	case middleware.AuthError:
		errorCode = http.StatusUnauthorized
		resp := ErrorResponse{}
		resp.Errors = append(resp.Errors, ResponseError{Message: err.Error()})
		resp.Ok = false
		errorResponse = resp
		w.Header().Set("WWW-Authenticate", "Basic")
	case middleware.ForbiddenError:
		errorCode = http.StatusForbidden
		resp := ErrorResponse{}
		resp.Errors = append(resp.Errors, ResponseError{Message: err.Error()})
		resp.Ok = false
		errorResponse = resp
	case ErrorResponse:
		errorCode = http.StatusBadRequest
		resp := err.(ErrorResponse)
		resp.Ok = false
		errorResponse = resp
	case BadRequestError:
		errorCode = http.StatusBadRequest
		resp := ErrorResponse{}
		resp.Errors = append(resp.Errors, ResponseError{Message: err.Error()})
		resp.Ok = false
		errorResponse = resp
	default:
		errorCode = http.StatusInternalServerError
		resp := ErrorResponse{}
		resp.Errors = append(resp.Errors, ResponseError{Message: "internal server error"})
		resp.Ok = false
		errorResponse = resp
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

// Error implements error interface
func (e ErrorResponse) Error() string {
	var errs []string
	for _, err := range e.Errors {
		errs = append(errs, err.Message)
	}
	return strings.Join(errs, ",")
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
/*
func (resp ClientResponse) Send(w http.ResponseWriter, r http.Request, code int) {
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
*/

/*

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
*/
