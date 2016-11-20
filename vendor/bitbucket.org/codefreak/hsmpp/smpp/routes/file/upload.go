package file

import (
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type uploadReq struct {
	Description string
	URL         string
	Token       string
}

type uploadResponse struct {
	ID string
}

// UploadHandler handles uploading of files
var UploadHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	maxPostSize := models.MaxFileSize + (1024 * 512)
	if r.ContentLength > maxPostSize {
		log.Error("Upload request too large.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "File size is too large. Upto 5 MB files are allowed.",
				},
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
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request form. Was it submitted as multipart?",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	err = routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing file upload request.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request",
				},
			},
			Request: uReq,
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

	nf := models.NumFile{
		Description: uReq.Description,
		Username:    u.Username,
		UserID:      u.ID,
		SubmittedAt: time.Now().UTC().Unix(),
	}
	f, h, err := r.FormFile("File")
	if err != nil {
		log.WithError(err).Error("Error getting file form field.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't get file.",
				},
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
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: err.Error(),
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.ID = id
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
