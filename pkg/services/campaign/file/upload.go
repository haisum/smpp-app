package file

import (
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign/file"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user"
	"bitbucket.org/codefreak/hsmpp/pkg/services"
	log "github.com/Sirupsen/logrus"
)

type uploadReq struct {
	Description string
	URL         string
	Token       string
}

type uploadResponse struct {
	ID int64
}

// UploadHandler handles uploading of files
var UploadHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	maxPostSize := file.MaxFileSize + (1024 * 512)
	if r.ContentLength > maxPostSize {
		log.Error("Upload request too large.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
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
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
					Message: "Couldn't parse request form. Was it submitted as multipart?",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	err = services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing file upload request.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
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
		u  user.User
		ok bool
	)
	if u, ok = services.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}

	nf := file.NumFile{
		Description: uReq.Description,
		Username:    u.Username,
		SubmittedAt: time.Now().UTC().Unix(),
	}
	f, h, err := r.FormFile("File")
	if err != nil {
		log.WithError(err).Error("Error getting file form field.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
					Message: "Couldn't get file.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	id, err := nf.Save(h.Filename, f, &file.RealNumFileIO{})
	if err != nil {
		log.WithError(err).Error("Error saving file.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeRequest,
					Message: err.Error(),
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.ID = id
	resp := services.ClientResponse{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
