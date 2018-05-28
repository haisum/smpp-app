package user

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/haisum/smpp-app/pkg/entities/user"
	"github.com/haisum/smpp-app/pkg/entities/user/permission"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/haisum/smpp-app/pkg/response"
	"github.com/haisum/smpp-app/pkg/services/middleware"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHandler returns a http handler for the user service.
func MakeHandler(svc Service, opts []kithttp.ServerOption, responseEncoder kithttp.EncodeResponseFunc) http.Handler {

	authMid := middleware.AuthMiddleware(svc.(*service).authenticator, "", "")
	infoHandler := kithttp.NewServer(
		authMid(makeInfoEndpoint(svc)),
		decodeInfoRequest,
		responseEncoder, opts...)
	editHandler := kithttp.NewServer(
		authMid(makeEditEndpoint(svc)),
		decodeEditRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()

	r.Handle("/user/v1/info", infoHandler).Methods("GET")
	r.Handle("/user/v1/edit", editHandler).Methods("POST")
	return r
}

type infoRequest struct {
	URL string
}

type infoResponse struct {
	Username        string
	Name            string
	Email           string
	ConnectionGroup string
	Permissions     []permission.Permission
	RegisteredAt    int64
	Suspended       bool
}

func makeInfoEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(infoRequest)
		v, err := svc.Info(ctx, req)
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

func decodeInfoRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request infoRequest
	request.URL = r.URL.RequestURI()
	return request, nil
}

type editRequest struct {
	URL      string
	Password string
	Name     string
	Email    string
}

type editResponse struct {
	User *user.User
}

func makeEditEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(editRequest)
		v, err := svc.Edit(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := response.Success{Obj: v}
		req.Password = ""
		resp.Request = req
		return resp, nil
	}
}

func decodeEditRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request editRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
