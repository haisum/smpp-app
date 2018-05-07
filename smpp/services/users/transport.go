package users

import (
	"context"
	"net/http"

	"encoding/json"

	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/services"
	"bitbucket.org/codefreak/hsmpp/smpp/services/middleware"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHandler returns a http handler for the user service.
func MakeHandler(svc Service, opts []kithttp.ServerOption, responseEncoder kithttp.EncodeResponseFunc) http.Handler {
	authenticator := svc.(*service).authenticator
	authMid := middleware.AuthMiddleware(authenticator, "", "")
	permissionsHandler := kithttp.NewServer(
		authMid(makePermissionsEndpoint(svc)),
		decodePermissionsRequest,
		responseEncoder, opts...)
	authMid = middleware.AuthMiddleware(authenticator, "", permission.EditUsers)
	editHandler := kithttp.NewServer(
		authMid(makeEditEndpoint(svc)),
		decodeEditRequest,
		responseEncoder, opts...)
	authMid = middleware.AuthMiddleware(authenticator, "", permission.AddUsers)
	addHandler := kithttp.NewServer(
		authMid(makeAddEndpoint(svc)),
		decodeAddRequest,
		responseEncoder, opts...)
	authMid = middleware.AuthMiddleware(authenticator, "", permission.ListUsers)
	listHandler := kithttp.NewServer(
		authMid(makeListEndpoint(svc)),
		decodeListRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()

	r.Handle("/users/v1/permissions", permissionsHandler).Methods("GET")
	r.Handle("/users/v1/add", addHandler).Methods("POST")
	r.Handle("/users/v1/edit", editHandler).Methods("POST")
	r.Handle("/users/v1/list", listHandler).Methods("GET", "POST")
	return r
}

type permissionsRequest struct {
	URL string
}

type permissionsResponse struct {
	Permissions []permission.Permission
}

func makePermissionsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(permissionsRequest)
		v, err := svc.Permissions(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := services.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodePermissionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request permissionsRequest
	request.URL = r.URL.RequestURI()
	return request, nil
}

type listRequest struct {
	user.Criteria
	URL string
}

type listResponse struct {
	Users []user.User
}

func decodeListRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request listRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
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
		resp := services.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

type editRequest struct {
	URL             string
	Username        string
	Password        string
	Permissions     []permission.Permission
	Name            string
	Email           string
	ConnectionGroup string
	Suspended       bool
}

type editResponse struct {
	User *user.User
}

func decodeEditRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request editRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
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
		resp := services.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

type addRequest struct {
	URL             string
	Username        string
	Password        string
	Permissions     []permission.Permission
	Name            string
	Email           string
	ConnectionGroup string
	Suspended       bool
}

type addResponse struct {
	ID int64
}

func decodeAddRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request addRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func makeAddEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addRequest)
		v, err := svc.Add(ctx, req)
		if err != nil {
			if errResponse, ok := err.(errs.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := services.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}
