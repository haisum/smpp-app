package file

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/numfile"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type downloadRequest struct {
	ID    int64
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
		u  user.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	files, err := numfile.List(numfile.Criteria{
		ID: uReq.ID,
	})

	if u.ID != files[0].UserID {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, permission.ListNumFiles); !ok {
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
	filepath := fmt.Sprintf("%s/%s/%s", numfile.Path, files[0].UserID, files[0].LocalName)
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
