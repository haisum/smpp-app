package file

import (
	"context"
	"encoding/json"
	"net/http"

	"bitbucket.org/codefreak/hsmpp/pkg/errs"
	"bitbucket.org/codefreak/hsmpp/pkg/response"
	"bitbucket.org/codefreak/hsmpp/pkg/services/middleware"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHandler returns a http handler for the message service.
func MakeHandler(svc Service, opts []kithttp.ServerOption, responseEncoder kithttp.EncodeResponseFunc) http.Handler {
	authMid := middleware.AuthMiddleware(svc.(*service).authenticator, "", "")
	deleteHandler := kithttp.NewServer(
		authMid(makeDeleteEndpoint(svc)),
		decodeDeleteRequest,
		responseEncoder, opts...)
	downloadHandler := kithttp.NewServer(
		authMid(makeDownloadEndpoint(svc)),
		decodeDownloadRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()
	r.Handle("/campaign/file/v1/delete", deleteHandler).Methods("POST")
	r.Handle("/campaign/file/v1/download", downloadHandler).Methods("GET")

	return r
}

type deleteRequest struct {
	URL string
	ID  int64
}

type deleteResponse struct {
}

func makeDeleteEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteRequest)
		v, err := svc.Delete(ctx, req)
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

func decodeDeleteRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type downloadRequest struct {
	ID    int64
	Token string
	URL   string
}

type downloadResponse struct {
}

func makeDownloadEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(downloadRequest)
		v, err := svc.Download(ctx, req)
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

func decodeDownloadRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request downloadRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
