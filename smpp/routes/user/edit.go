package user

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type editRequest struct {
	URL      string
	Token    string
	Password string
	Name     string
	Email    string
}

type editResponse struct {
	User user.User
}

//EditHandler allows editing a user
var EditHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := editResponse{}
	var uReq editRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing user edit request.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	var (
		u  user.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	log.WithField("user", u).Info("user")

	if uReq.Name != "" {
		u.Name = uReq.Name
	}
	if uReq.Email != "" {
		u.Email = uReq.Email
	}
	if uReq.Password != "" {
		u.Password = uReq.Password
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
	}
	verrs, err := u.Validate()
	if err != nil {
		resp.Ok = false
		resp.Errors = make([]routes.ResponseError, len(verrs))
		for k, v := range verrs {
			resp.Errors = append(resp.Errors, routes.ResponseError{
				Type:    routes.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	err = u.Update(len(uReq.Password) > 1)
	if err != nil {
		log.WithError(err).Error("Couldn't update user.")
		resp = routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeDB,
					Message: "Couldn't update user",
				},
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.User = u
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
