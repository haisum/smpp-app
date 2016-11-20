package users

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type editRequest struct {
	URL             string
	Token           string
	Username        string
	Password        string
	Permissions     []user.Permission
	Name            string
	Email           string
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, user.PermEditUsers); !ok {
		return
	}
	u, err := models.GetUser(uReq.Username)
	if err != nil {
		log.WithError(err).Error("Error getting user.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't get user.",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	log.WithField("user", u).Info("user")

	if uReq.Name != "" {
		u.Name = uReq.Name
	}
	if uReq.Email != "" {
		u.Email = uReq.Email
	}
	if uReq.ConnectionGroup != "" {
		u.ConnectionGroup = uReq.ConnectionGroup
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
	}
	verrs, err := u.Validate()
	if err != nil {
		resp.Ok = false
		resp.Errors = make([]routes.ResponseError, len(verrs))
		for k, v := range verrs {
			resp.Errors = append(resp.Errors, routes.ResponseError{
				Type:    routes.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	err = u.Update(len(uReq.Password) > 1)
	if err != nil {
		log.WithError(err).Error("Couldn't update user.")
		resp = routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeDB,
					Message: "Couldn't update user",
				},
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
