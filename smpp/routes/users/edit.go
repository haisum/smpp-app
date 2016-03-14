package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type editRequest struct {
	Url             string
	AuthToken       string
	Username        string
	Password        string
	Permissions     []models.Permission
	Name            string
	Email           string
	NightStartAt    string
	NightEndAt      string
	ConnectionGroup string
	Suspended       bool
}

type editResponse struct {
	User models.User
}

// Edit handler allows editing a user
func Edit(w http.ResponseWriter, r *http.Request) {
	uResp := editResponse{}
	var uReq editRequest
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
		log.WithError(err).Error("Error parsing user edit request.")
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
	u, err := models.GetUser(s, uReq.Username)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't get user with that username",
				"db": err.Error(),
			},
		}
		log.WithError(err).Error("Couldn't get user.")
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

	if uReq.Name != "" {
		u.Name = uReq.Name
	}
	if uReq.NightEndAt != "" {
		u.NightEndAt = uReq.NightEndAt
	}
	if uReq.Email != "" {
		u.Email = uReq.Email
	}
	if uReq.ConnectionGroup != "" {
		u.ConnectionGroup = uReq.ConnectionGroup
	}
	if uReq.NightStartAt != "" {
		u.NightStartAt = uReq.NightStartAt
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
		Ok:      true,
	}
	verrs, err := u.Validate()
	if err != nil {
		resp.Ok = false
		resp.Errors = verrs
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
	err = u.Update(s, len(uReq.Password) > 1)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusInternalServerError): "Couldn't update user",
				"db": err.Error(),
			},
		}
		log.WithError(err).Error("Couldn't update user.")
		b, cType, err := routes.MakeResponse(*r, resp)
		if err != nil {
			log.WithError(err).Error("Couldn't make response.")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}
	uResp.User = u
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
