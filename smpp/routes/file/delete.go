package file

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type deleteRequest struct {
	Url   string
	Token string
	Id    string
}

type deleteResponse struct {
}

// MessagesHandler allows adding a user to database
var DeleteHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := deleteResponse{}
	var uReq deleteRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing delete request.")
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
	uReq.Url = r.URL.RequestURI()
	var (
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	files, err := models.GetNumFiles(models.NumFileCriteria{
		Id: uReq.Id,
	})
	resp := routes.Response{}
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
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermDeleteNumFile); !ok {
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
