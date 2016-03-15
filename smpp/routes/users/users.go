package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type usersRequest struct {
	models.UserCriteria
	Url string
}

type usersResponse struct {
	Users []models.User
}

// Add handler allows adding a user to database
func Users(w http.ResponseWriter, r *http.Request) {
	uResp := usersResponse{}
	var uReq usersRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse request",
			},
		}
		b, cType, err := routes.MakeResponse(*r, resp)
		if err != nil {
			log.WithError(err).Error("Couldn't make response.")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		log.WithError(err).Error("Error parsing users list request.")
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
	log.Infof("Ureq %+v", uReq)
	uReq.Url = r.URL.RequestURI()
	s, err := db.GetSession()
	if err != nil {
		resp := routes.Response{}
		resp.Ok = false
		log.WithError(err).Error("Error in getting session.")
		resp.Errors = routes.ResponseErrors{"db": "Couldn't connect to database."}
		resp.Request = uReq
		b, cType, err := routes.MakeResponse(*r, resp)
		if err != nil {
			log.WithError(err).Error("Couldn't create response.")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}
	users, err := models.GetUsers(s, uReq.UserCriteria)
	resp := routes.Response{}
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"db": "Couldn't get users."}
		resp.Request = uReq
		b, cType, err := routes.MakeResponse(*r, resp)
		if err != nil {
			log.WithError(err).Error("Couldn't make response.")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
	uResp.Users = users
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	b, cType, err := routes.MakeResponse(*r, resp)
	if err != nil {
		log.WithError(err).Error("Couldn't make response.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", cType)
	fmt.Fprint(w, string(b))
}
