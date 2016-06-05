package services

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/supervisor/services"
	log "github.com/Sirupsen/logrus"
)

type statusRequest struct {
	URL   string
	Token string
}

// StatusHandler sees supervisorctl and returns status of all running processes
var StatusHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq getConfigRequest
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermShowConfig); !ok {
		return
	}
	st, err := services.GetStatus()
	if err != nil {
		log.WithError(err).Error("Couldn't get status")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "Couldn't get status.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp := routes.Response{}

	resp.Obj = st
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
