package errs

import (
	"fmt"
	"net/http"

	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/response"
	"github.com/pkg/errors"
)

// ValidationError is returned when data validation fails
type ValidationError struct {
	Errors  map[string]string
	Message string
}

// Error implements Error interface
func (v *ValidationError) Error() string {
	return v.Message
}

// BadRequestError is sent when user sends invalid request
type BadRequestError error

// ErrHandler is function called to log errors anywhere in application
func ErrHandler(err error) {
	go func() {
		cause := errors.Cause(err)
		logger.Get().Error("type", fmt.Sprintf("%T", cause), "cause", cause, "error", fmt.Sprintf("%s", err), "stackTrace", fmt.Sprintf("%+v", err))
	}()
}

// ErrResponseHandler handles error returned to browser in case something goes wrong during request
func ErrResponseHandler(err error) (errorResponse error, errorCode int) {
	err = errors.Cause(err)
	switch err.(type) {
	case AuthError:
		errorCode = http.StatusUnauthorized
		resp := ErrorResponse{}
		resp.Errors = append(resp.Errors, ResponseError{Message: err.Error()})
		resp.Ok = false
		errorResponse = resp
	case ForbiddenError:
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
	return
}

// ErrorResponse is sent when Error happens in request
type ErrorResponse struct {
	Errors []ResponseError
	Cause  error `xml:"-" json:"-"`
	response.Response
}

// AuthErrorResponse is sent when authentication error has happened
type AuthErrorResponse struct {
	ErrorResponse
}

// ForbiddenErrorResponse is sent when access to a resource is forbidden
// usually because of permissions
type ForbiddenErrorResponse struct {
	ErrorResponse
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
	ErrorTypeQueue   string = "queue"
	ErrorTypeRequest string = "request"
	ErrorTypeConfig  string = "config"
)

// AuthError represents an authorization error.
type AuthError struct {
}

// Error is an implementation of the Error interface.
func (AuthError) Error() string {
	return http.StatusText(http.StatusUnauthorized)
}

// ForbiddenError represents an authorization error.
type ForbiddenError struct {
	Message string
}

// StatusCode is an implementation of the StatusCoder interface in go-kit/http.
func (ForbiddenError) StatusCode() int {
	return http.StatusForbidden
}

// Error is an implementation of the Error interface.
func (f ForbiddenError) Error() string {
	if f.Message != "" {
		return f.Message
	}
	return http.StatusText(http.StatusForbidden)
}
