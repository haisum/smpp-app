package users

import (
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type addRequest struct {
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

type addResponse struct {
	ID string
}

// AddHandler allows adding a user to database
var AddHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := addResponse{}
	var uReq addRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing user add request.")
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, user.PermAddUsers); !ok {
		return
	}
	u := models.User{
		Email:           uReq.Email,
		ConnectionGroup: uReq.ConnectionGroup,
		Username:        uReq.Username,
		Password:        uReq.Password,
		Name:            uReq.Name,
		Permissions:     uReq.Permissions,
		RegisteredAt:    time.Now().UTC().Unix(),
		Suspended:       uReq.Suspended,
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
	id, err := u.Add()
	if err != nil {
		log.WithError(err).Error("Couldn't add user.")
		resp = routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeDB,
					Message: "Couldn't add user",
				},
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.ID = id
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
