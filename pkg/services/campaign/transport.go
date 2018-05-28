package campaign

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/haisum/smpp-app/pkg/entities/campaign"
	"github.com/haisum/smpp-app/pkg/entities/user/permission"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/haisum/smpp-app/pkg/response"
	"github.com/haisum/smpp-app/pkg/services/middleware"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHandler returns a http handler for the message service.
func MakeHandler(svc Service, opts []kithttp.ServerOption, responseEncoder kithttp.EncodeResponseFunc) http.Handler {
	authMid := middleware.AuthMiddleware(svc.(*service).authenticator, "", "")
	listHandler := kithttp.NewServer(
		authMid(makeListEndpoint(svc)),
		decodeListRequest,
		responseEncoder, opts...)
	authMid = middleware.AuthMiddleware(svc.(*service).authenticator, "", permission.StartCampaign)
	startHandler := kithttp.NewServer(
		authMid(makeStartEndpoint(svc)),
		decodeStartRequest,
		responseEncoder, opts...)
	progressHandler := kithttp.NewServer(
		authMid(makeProgressEndpoint(svc)),
		decodeProgressRequest,
		responseEncoder, opts...)
	reportHandler := kithttp.NewServer(
		authMid(makeReportEndpoint(svc)),
		decodeReportRequest,
		responseEncoder, opts...)
	authMid = middleware.AuthMiddleware(svc.(*service).authenticator, "", permission.StopCampaign)
	stopHandler := kithttp.NewServer(
		authMid(makeStopEndpoint(svc)),
		decodeStopRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()

	r.Handle("/campaign/v1/list", listHandler).Methods("GET", "POST")
	r.Handle("/campaign/v1/start", startHandler).Methods("POST")
	r.Handle("/campaign/v1/progress", progressHandler).Methods("GET", "POST")
	r.Handle("/campaign/v1/stop", stopHandler).Methods("POST")
	r.Handle("/campaign/v1/report", reportHandler).Methods("GET", "POST")
	return r
}

type stopRequest struct {
	CampaignID int64
	URL        string
}

type stopResponse struct {
	Count int64
}

func makeStopEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(stopRequest)
		v, err := svc.Stop(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := response.Success{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeStopRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request stopRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type reportRequest struct {
	CampaignID int64
	URL        string
}

type reportResponse struct {
	campaign.Report
}

func makeReportEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(reportRequest)
		v, err := svc.Report(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := response.Success{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeReportRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request reportRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type progressRequest struct {
	CampaignID int64
	URL        string
}

type progressResponse struct {
	campaign.Progress
}

func makeProgressEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(progressRequest)
		v, err := svc.Progress(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := response.Success{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeProgressRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request progressRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type listRequest struct {
	campaign.Criteria
	URL string
}

type listResponse struct {
	Campaigns []campaign.Campaign
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
		resp := response.Success{Obj: v}
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

type startRequest struct {
	URL         string
	FileID      int64
	Numbers     string
	Description string
	Priority    int
	Src         string
	Msg         string
	ScheduledAt int64
	SendBefore  string
	SendAfter   string
	Mask        bool
	IsFlash     bool
}

func (request *startRequest) validate() []errs.ResponseError {
	var errors []errs.ResponseError
	if request.Msg == "" {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "Msg",
			Message: "Can't send empty message.",
		})
	}
	if request.Description == "" {
		errors = append(errors, errs.ResponseError{
			Type:    errs.ErrorTypeForm,
			Field:   "Description",
			Message: "Description must be provided for campaign.",
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
	if request.SendAfter != "" {
		parts := strings.Split(request.SendAfter, ":")
		if len(parts) != 2 {
			errors = append(errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Field:   "SendAfter",
				Message: "Send after must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 {
				errors = append(errors, errs.ResponseError{
					Type:    errs.ErrorTypeForm,
					Field:   "SendAfter",
					Message: "Send after must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	if request.SendBefore != "" {
		parts := strings.Split(request.SendBefore, ":")
		if len(parts) != 2 {
			errors = append(errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Field:   "SendBefore",
				Message: "Send before must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 {
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

type startResponse struct {
	ID int64
}

func makeStartEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(startRequest)
		v, err := svc.Start(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := response.Success{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeStartRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request startRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
