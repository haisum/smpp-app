package services

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type getConfigRequest struct {
	URL   string
	Token string
}

// GetConfigHandler gets invoked on a get request from user and returns
// current configuration in database.
var GetConfigHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq getConfigRequest
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, permission.ShowConfig); !ok {
		return
	}
	c, err := smpp.GetConfig()
	if err != nil {
		log.WithError(err).Error("Couldn't get config")
		resp := routes.ClientResponse{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "Couldn't get config.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp := routes.ClientResponse{}

	resp.Obj = c
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
