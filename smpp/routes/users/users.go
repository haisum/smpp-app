package users

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type usersRequest struct {
	models.UserCriteria
	Url   string
	Token string
}

type usersResponse struct {
	Users []models.User
}

// UsersHandler allows adding a user to database
var UsersHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := usersResponse{}
	var uReq usersRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing users list request.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse request",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.Url = r.URL.RequestURI()
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermListUsers); !ok {
		return
	}
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Error in getting session.")
		resp := routes.Response{}
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"db": "Couldn't connect to database."}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	users, err := models.GetUsers(s, uReq.UserCriteria)
	resp := routes.Response{}
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"db": "Couldn't get users."}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.Users = users
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
