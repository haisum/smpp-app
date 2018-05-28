package file

import (
	"context"
	"encoding/json"
	"net/http"

	"io"

	"fmt"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/haisum/smpp-app/pkg/entities/campaign/file"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/haisum/smpp-app/pkg/response"
	"github.com/haisum/smpp-app/pkg/services/middleware"
	"github.com/pkg/errors"
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
	uploadHandler := kithttp.NewServer(
		authMid(makeUploadEndpoint(svc)),
		decodeUploadRequest,
		responseEncoder, opts...)
	listHandler := kithttp.NewServer(
		authMid(makeListEndpoint(svc)),
		decodeListRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()
	r.Handle("/campaign/file/v1/delete", deleteHandler).Methods("POST")
	r.Handle("/campaign/file/v1/download", downloadHandler).Methods("GET")
	r.Handle("/campaign/file/v1/upload", uploadHandler).Methods("POST")
	r.Handle("/campaign/file/v1/list", listHandler).Methods("POST", "GET")
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

type listRequest struct {
	file.Criteria
	URL string
}

type listResponse struct {
	Files []file.File
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
		return v, nil
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

type uploadRequest struct {
	FileName    string
	Description string
	URL         string
	ReadCloser  io.ReadCloser `json:"-"`
}

type uploadResponse struct {
	ID int64
}

func makeUploadEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(uploadRequest)
		v, err := svc.Upload(ctx, req)
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

func decodeUploadRequest(_ context.Context, r *http.Request) (interface{}, error) {
	request := uploadRequest{}
	request.URL = r.URL.RequestURI()
	maxPostSize := file.MaxFileSize + (1024 * 512)
	if r.ContentLength > maxPostSize {
		return request, fmt.Errorf("file size can't be larger than %d", file.MaxFileSize)
	}
	err := r.ParseMultipartForm(maxPostSize)
	if err != nil {
		return request, errors.Wrap(err, "error in parsing multi part form, check size")
	}
	f, h, err := r.FormFile("File")

	request.ReadCloser = f
	request.FileName = h.Filename
	request.Description = r.PostFormValue("Description")
	request.URL = r.URL.RequestURI()

	return request, nil
}
