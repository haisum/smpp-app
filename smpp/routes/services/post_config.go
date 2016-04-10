package services

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type postConfigRequest struct {
	Url    string
	Token  string
	Config smpp.Config
}

// PostConfigHandler gets invoked on a post request from user and
// saves supplied config in database
var PostConfigHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq postConfigRequest
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
	if !routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermEditConfig) {
		return
	}
	err = models.SetConfig(uReq.Config)
	if err != nil {
		log.WithError(err).Error("Couldn't set config.")
		resp := routes.Response{
			Errors:  routes.ResponseErrors{"config": "Couldn't set config."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	resp := routes.Response{}
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
