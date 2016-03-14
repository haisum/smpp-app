package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"time"
)

type addRequest struct {
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

type addResponse struct {
	Id string
}

// Add handler allows adding a user to database
func Add(w http.ResponseWriter, r *http.Request) {
	uResp := addResponse{}
	var uReq addRequest
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
		log.WithError(err).Error("Error parsing user add request.")
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
	u := models.User{
		Email:           uReq.Email,
		ConnectionGroup: uReq.ConnectionGroup,
		NightEndAt:      uReq.NightEndAt,
		NightStartAt:    uReq.NightStartAt,
		Username:        uReq.Username,
		Password:        uReq.Password,
		Name:            uReq.Name,
		Permissions:     uReq.Permissions,
		RegisteredAt:    time.Now().Unix(),
		Suspended:       uReq.Suspended,
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
	id, err := u.Add(s)
	if err != nil {
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusInternalServerError): "Couldn't add user",
				"db": err.Error(),
			},
		}
		log.WithError(err).Error("Couldn't add user.")
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
	uResp.Id = id
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
