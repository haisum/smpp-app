package user

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"net/http"
)

type infoRequest struct {
	Url   string
	Token string
}

type infoResponse struct {
	Username        string
	Name            string
	Email           string
	NightStartAt    string
	NightEndAt      string
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
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse auth request",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.Url = r.URL.RequestURI()
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
	uResp.NightEndAt = u.NightEndAt
	uResp.NightStartAt = u.NightStartAt
	uResp.NightEndAt = u.NightEndAt
	uResp.Suspended = u.Suspended
	uResp.RegisteredAt = u.RegisteredAt
	uResp.Username = u.Username
	uResp.Name = u.Name

	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
