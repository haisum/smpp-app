package file

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models/campaign/file"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/services"
	log "github.com/Sirupsen/logrus"
)

type filterRequest struct {
	file.Criteria
	URL   string
	Token string
}

type filterResponse struct {
	NumFiles []file.NumFile
}

// FilterHandler searches files in NumFiles table
var FilterHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := filterResponse{}
	var uReq filterRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing files list request.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
					Message: "Couldn't parse request.",
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
	if u, ok = services.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	if u.Username != uReq.Username {
		if _, ok = services.Authenticate(w, *r, uReq, uReq.Token, permission.ListNumFiles); !ok {
			return
		}
	}
	files, err := file.List(uReq.Criteria)
	resp := services.ClientResponse{}
	if err != nil {
		resp.Ok = false
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeDB,
				Message: "Couldn't get files.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.NumFiles = files
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
