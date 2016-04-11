package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type editRequest struct {
	Url             string
	Token           string
	Username        string
	Password        string
	Permissions     []smpp.Permission
	Name            string
	Email           string
	NightStartAt    string
	NightEndAt      string
	ConnectionGroup string
	Suspended       bool
}

type editResponse struct {
	User models.User
}

//EditHandler allows editing a user
var EditHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := editResponse{}
	var uReq editRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing user edit request.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse request",
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
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermEditUsers); !ok {
		return
	}
	log.WithField("user", u).Info("user")

	if uReq.Name != "" {
		u.Name = uReq.Name
	}
	if uReq.NightEndAt != "" {
		u.NightEndAt = uReq.NightEndAt
	}
	if uReq.Email != "" {
		u.Email = uReq.Email
	}
	if uReq.ConnectionGroup != "" {
		u.ConnectionGroup = uReq.ConnectionGroup
	}
	if uReq.NightStartAt != "" {
		u.NightStartAt = uReq.NightStartAt
	}
	if uReq.Password != "" {
		u.Password = uReq.Password
	}
	if len(uReq.Permissions) > 0 {
		u.Permissions = uReq.Permissions
	}
	if uReq.Suspended == true {
		u.Suspended = true
	}
	if u.Suspended == true && uReq.Suspended == false {
		u.Suspended = false
	}

	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	verrs, err := u.Validate()
	if err != nil {
		resp.Ok = false
		resp.Errors = verrs
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	s, _ := db.GetSession()
	err = u.Update(s, len(uReq.Password) > 1)
	if err != nil {
		log.WithError(err).Error("Couldn't update user.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusInternalServerError): "Couldn't update user",
				"db": err.Error(),
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.User = u
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
