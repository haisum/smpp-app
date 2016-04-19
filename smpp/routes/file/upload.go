package file

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"time"
)

type uploadReq struct {
	Name        string
	Description string
	Url         string
	Token       string
}

type uploadResponse struct {
	Id string
}

// MessageHandler allows sending one sms
var UploadHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	maxPostSize := models.MaxFileSize + (1024 * 512)
	if r.ContentLength > maxPostSize {
		log.Error("Upload request too large.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"request": "File size is too large. Upto 5 MB files are allowed.",
			},
		}
		resp.Send(w, *r, http.StatusExpectationFailed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPostSize)
	uResp := uploadResponse{}
	var uReq uploadReq
	err := r.ParseMultipartForm(maxPostSize)
	if err != nil {
		log.WithError(err).Error("Error parsing multipart form.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"request": "Couldn't parse request form. Was it submitted as multipart?",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	err = routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing file upload request.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"request": "Couldn't parse request",
			},
			Request: uReq,
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

	nf := models.NumFile{
		Name:        uReq.Name,
		Description: uReq.Description,
		Username:    u.Username,
		UserId:      u.Id,
		SubmittedAt: time.Now().Unix(),
	}
	f, h, err := r.FormFile("File")
	if err != nil {
		log.WithError(err).Error("Error getting file form field.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"request": "Couldn't get File.",
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	id, err := nf.Save(h.Filename, f)
	if err != nil {
		log.WithError(err).Error("Error saving file.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				"request": err.Error(),
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.Id = id
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
