package user

import (
	"context"
	"encoding/json"
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user/permission"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// MakeHandler returns a http handler for the user service.
func MakeHandler(svc Service, errEncoder kithttp.ErrorEncoder, responseEncoder kithttp.EncodeResponseFunc) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(errEncoder),
	}
	tokenHandler := kithttp.NewServer(
		makeTokenEndpoint(svc),
		decodeTokenRequest,
		responseEncoder, opts...)
	infoHandler := kithttp.NewServer(
		makeInfoEndpoint(svc),
		decodeInfoRequest,
		responseEncoder, opts...)
	editHandler := kithttp.NewServer(
		makeEditEndpoint(svc),
		decodeEditRequest,
		responseEncoder, opts...)
	r := mux.NewRouter()

	r.Handle("/user/v1/info", infoHandler).Methods("GET")
	r.Handle("/user/v1/token", tokenHandler).Methods("POST")
	r.Handle("/user/v1/edit", editHandler).Methods("POST")
	return r
}

type tokenRequest struct {
	URL      string
	Username string
	Password string
	Validity int
}

type tokenResponse struct {
	Token string
}

func makeTokenEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(tokenRequest)
		v, err := svc.Token(ctx, req)
		req.Password = ""
		if err != nil {
			if errResponse, ok := err.(routes.AuthErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			if errResponse, ok := err.(routes.ForbiddenErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		// make sure we aren't sending back password
		resp := routes.SuccessResponse{Obj: v}
		resp.Request = req
		return resp, nil
	}
}

func decodeTokenRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request tokenRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, errors.Wrap(err, "decode token error")
	}
	return request, nil
}

type infoRequest struct {
	URL   string
	Token string
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
			if errResponse, ok := err.(routes.ErrorResponse); ok {
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

func decodeInfoRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request infoRequest
	request.URL = r.URL.RequestURI()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

type editRequest struct {
	URL      string
	Token    string
	Password string
	Name     string
	Email    string
}

type editResponse struct {
	User *User
}

func makeEditEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(editRequest)
		v, err := svc.Edit(ctx, req)
		if err != nil {
			if errResponse, ok := err.(routes.ErrorResponse); ok {
				errResponse.Response.Request = req
				return nil, errResponse
			}
			return nil, err
		}
		resp := routes.SuccessResponse{Obj: v}
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
