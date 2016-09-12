package user

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type authRequest struct {
	URL      string
	Username string
	Password string
	Validity int
}

type authResponse struct {
	Token string
}

// AuthHandler  returns a token against valid username/password pair
var AuthHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := authResponse{}
	var uReq authRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request.",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	u, err := models.GetUser(uReq.Username)
	if err != nil {
		resp.Ok = false
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeAuth,
				Message: "Username/Password pair is wrong.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	if !u.Auth(uReq.Password) {
		log.WithError(err).Error("Couldn't authenticate user.")
		resp = routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeAuth,
					Message: "Username/password pair is wrong.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	if u.Suspended {
		resp = routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeAuth,
					Message: "User is suspended.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	token, _ := models.CreateToken(u.Username, uReq.Validity)
	uResp.Token = token
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
