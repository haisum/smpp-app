package file

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type downloadRequest struct {
	ID    string
	Token string
	URL   string
}

// DownloadHandler downloads uploaded numfile
var DownloadHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	var uReq downloadRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing download request.")
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
	uReq.URL = r.URL.RequestURI()
	var (
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	files, err := models.GetNumFiles(models.NumFileCriteria{
		ID: uReq.ID,
	})

	if u.ID != files[0].UserID {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, user.PermListNumFiles); !ok {
			return
		}
	}
	resp := routes.Response{}
	if err != nil || len(files) == 0 {
		log.WithFields(log.Fields{
			"len(files)": len(files),
			"Error":      err,
			"Username":   files[0].Username,
		}).Error("Error finding file from db")
		resp.Ok = false
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get files.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	filepath := fmt.Sprintf("%s/%s/%s", models.NumFilePath, files[0].UserID, files[0].LocalName)
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.WithFields(log.Fields{
			"filepath": filepath,
			"Error":    err,
		}).Error("Error reading file")
		resp.Ok = false
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
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
