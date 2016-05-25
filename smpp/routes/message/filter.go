package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type messagesRequest struct {
	models.MessageCriteria
	Url   string
	Token string
}

type messagesResponse struct {
	Messages []models.Message
	Stats    models.MessageStats
}

// MessagesHandler allows adding a user to database
var MessagesHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := messagesResponse{}
	var uReq messagesRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing messages list request.")
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
	if u.Username != uReq.Username {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermListMessages); !ok {
			return
		}
	}
	messages, err := models.GetMessages(uReq.MessageCriteria)
	resp := routes.Response{}
	if err != nil {
		resp.Ok = false
		log.WithError(err).Error("Couldn't get message.")
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get messages.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	stats, err := models.GetMessageStats(uReq.MessageCriteria)
	if err != nil {
		resp.Ok = false
		log.WithError(err).Error("Couldn't get message stats.")
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get message stats.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Messages = messages
	uResp.Stats = stats
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
