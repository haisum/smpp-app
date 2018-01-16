package services

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/supervisor"
	log "github.com/Sirupsen/logrus"
)

type postConfigRequest struct {
	URL    string
	Token  string
	Config smpp.Config
}

// PostConfigHandler gets invoked on a post request from user and
// saves supplied config in database
var PostConfigHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq postConfigRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		resp := routes.ClientResponse{
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, permission.EditConfig); !ok {
		return
	}
	err = smpp.SetConfig(uReq.Config)
	if err != nil {
		log.WithError(err).Error("Couldn't set config.")
		resp := routes.ClientResponse{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "Couldn't set configuration.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	_, err = supervisor.Execute("reload")
	if err != nil {
		log.WithError(err).Error("Couldn't reload supervisor.")
		resp := routes.ClientResponse{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "Couldn't reload supervisor.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp := routes.ClientResponse{}
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
