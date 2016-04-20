package file

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type filterRequest struct {
	models.NumFileCriteria
	Url   string
	Token string
}

type filterResponse struct {
	NumFiles []models.NumFile
}

// FilterHandler searches files in NumFiles table
var FilterHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := filterResponse{}
	var uReq filterRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing files list request.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse request",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.Url = r.URL.RequestURI()
	var (
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	if u.Username != uReq.Username {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermListNumFiles); !ok {
			return
		}
	}
	files, err := models.GetNumFiles(uReq.NumFileCriteria)
	resp := routes.Response{}
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"db": "Couldn't get files."}
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
