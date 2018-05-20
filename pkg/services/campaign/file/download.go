package file

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign/file"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/services"
	log "github.com/Sirupsen/logrus"
)

type downloadRequest struct {
	ID    int64
	Token string
	URL   string
}

// DownloadHandler downloads uploaded file
var DownloadHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq downloadRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing download request.")
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
	files, err := file.List(file.Criteria{
		ID: uReq.ID,
	})

	if u.Username != files[0].Username {
		if _, ok = services.Authenticate(w, *r, uReq, uReq.Token, permission.ListNumFiles); !ok {
			return
		}
	}
	resp := services.ClientResponse{}
	if err != nil || len(files) == 0 {
		log.WithFields(log.Fields{
			"len(files)": len(files),
			"Error":      err,
			"Username":   files[0].Username,
		}).Error("Error finding file from db")
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
	filepath := fmt.Sprintf("%s/%s/%s", file.Path, files[0].Username, files[0].LocalName)
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.WithFields(log.Fields{
			"filepath": filepath,
			"Error":    err,
		}).Error("Error reading file")
		resp.Ok = false
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeDB,
				Message: "Couldn't get file from file system.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+files[0].Name)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(b)
})
