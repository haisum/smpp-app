package user

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
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
	if !routes.Authenticate(w, *r, uReq, uReq.Token, "") {
		return
	}
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Error in getting session.")
		resp := routes.Response{
			Ok:      false,
			Errors:  routes.ResponseErrors{"db": "Couldn't connect to database."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	if !routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermAddUsers) {
		return
	}
	t, err := models.GetToken(s, uReq.Token)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"auth": "Couldn't verify token.",
			},
			Request: uReq,
		}
		log.WithError(err).Error("Error in verifying token.")
		resp.Send(w, *r, http.StatusForbidden)
		return
	}

	resp := routes.Response{}
	u, err := models.GetUser(s, t.Username)
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"auth": "Couldn't get user associated with token."}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusForbidden)
		return
	}
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
