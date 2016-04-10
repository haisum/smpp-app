package services

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type getConfigRequest struct {
	Url   string
	Token string
}

// GetConfigHandler gets invoked on a get request from user and returns
// current configuration in database.
var GetConfigHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq getConfigRequest
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
	if !routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermShowConfig) {
		return
	}
	c, err := models.GetConfig()
	if err != nil {
		log.WithError(err).Error("Couldn't get config")
		resp := routes.Response{
			Errors:  routes.ResponseErrors{"db": "Couldn't get config."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp := routes.Response{}

	resp.Obj = c
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
