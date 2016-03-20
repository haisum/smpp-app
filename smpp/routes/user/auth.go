package user

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type authRequest struct {
	Url      string
	Username string
	Password string
}

type authResponse struct {
	Token string
}

// Auth handler returns a token against valid username/password pair
func Auth(w http.ResponseWriter, r *http.Request) {
	uResp := authResponse{}
	var uReq authRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse auth request",
			},
		}
		b, cType, err := routes.MakeResponse(*r, resp)
		if err != nil {
			log.WithError(err).Error("Couldn't make response.")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		log.WithError(err).Error("Error parsing user auth request.")
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
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
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	u, err := models.GetUser(s, uReq.Username)
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"auth": "Username/Password pair is wrong."}
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
	if !u.Auth(uReq.Password) {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"auth": "Username/Password pair is wrong.",
			},
		}
		log.WithError(err).Error("Couldn't authenticate user.")
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
	token, _ := models.CreateToken(s, u.Username)
	uResp.Token = token
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
