package user

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
)

type infoRequest struct {
	URL   string
	Token string
}

type infoResponse struct {
	Username        string
	Name            string
	Email           string
	ConnectionGroup string
	Permissions     []smpp.Permission
	RegisteredAt    int64
	Suspended       bool
}

// InfoHandler returns info of user, we are passing token for
var InfoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := infoResponse{}
	var uReq infoRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	var (
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	resp := routes.Response{}

	uResp.ConnectionGroup = u.ConnectionGroup
	uResp.Permissions = u.Permissions
	uResp.Suspended = u.Suspended
	uResp.RegisteredAt = u.RegisteredAt
	uResp.Username = u.Username
	uResp.Name = u.Name

	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
