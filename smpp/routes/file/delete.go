package file

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models/numfile"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type deleteRequest struct {
	URL   string
	Token string
	ID    int64
}

type deleteResponse struct {
}

// DeleteHandler marks a particular file deleted
var DeleteHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := deleteResponse{}
	var uReq deleteRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing delete request.")
		resp := routes.ClientResponse{
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
	var (
		u  user.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	files, err := numfile.List(numfile.Criteria{
		ID: uReq.ID,
	})
	resp := routes.ClientResponse{}
	if len(files) == 0 {
		resp.Ok = false
		log.WithError(err).Error("Couldn't get files.")
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get files.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	} else if files[0].Username != u.Username {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, permission.DeleteNumFile); !ok {
			return
		}
	}
	err = files[0].Delete()
	if err != nil {
		log.WithError(err).Error("Couldn't delete file")
		resp.Ok = false
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't delete file.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
