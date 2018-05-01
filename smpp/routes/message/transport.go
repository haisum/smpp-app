package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp/entities/message"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"context"
	"encoding/json"
	"github.com/go-kit/kit/endpoint"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type listRequest struct {
	message.Criteria
	URL string
}

type listResponse struct {
	Messages []message.Message
}

func makeListEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRequest)
		v, err := svc.List(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := routes.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeListRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request listRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type listDownloadRequest struct {
	listRequest
	// comma separated list of columns to populate
	ReportCols string
	TZ         string
}

func makeListDownloadEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listDownloadRequest)
		v, err := svc.ListDownload(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		return v, nil
	}
}

func decodeListDownloadRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request listDownloadRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type statsRequest struct {
	listRequest
}

type statsResponse struct {
	Stats message.Stats
}

func makeStatsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(statsRequest)
		v, err := svc.Stats(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := routes.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeStatsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request statsRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type sendRequest struct {
	Priority    int
	Src         string
	Dst         string
	Msg         string
	URL         string
	ScheduledAt int64
	IsFlash     bool
	SendBefore  string
	SendAfter   string
	Mask        bool
}

func (request *sendRequest) validate() []errs.ResponseError {
	var errors []errs.ResponseError
	if request.Dst == "" {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "Dst",
			Message: "Destination can't be empty.",
		})
	}
	if request.Msg == "" {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "Msg",
			Message: "Can't send empty message",
		})
	}
	if request.Src == "" {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "Src",
			Message: "Source address can't be empty.",
		})
	}
	if (request.SendAfter == "" && request.SendBefore != "") || (request.SendBefore == "" && request.SendAfter != "") {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeRequest,
			Message: "Send before time and Send after time, both should be provided at a time.",
		})
	}
	parts := strings.Split(request.SendAfter, ":")
	if request.SendAfter != "" {
		if len(parts) != 2 {
			errors = append(errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Field:   "SendAfter",
				Message: "Send after must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				errors = append(errors, errs.ResponseError{
					Type:    errs.ErrorTypeForm,
					Field:   "SendAfter",
					Message: "Send after must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	parts = strings.Split(request.SendBefore, ":")
	if request.SendBefore != "" {
		if len(parts) != 2 {
			errors = append(errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Field:   "SendBefore",
				Message: "Send before must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				errors = append(errors, errs.ResponseError{
					Type:    errs.ErrorTypeForm,
					Field:   "SendBefore",
					Message: "Send before must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	if request.ScheduledAt != 0 && request.ScheduledAt < time.Now().UTC().Unix() {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "ScheduledAt",
			Message: "Schedule time must be in future.",
		})
	}
	return errors
}

type sendResponse struct {
	ID int64
}
