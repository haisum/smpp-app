package campaign

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
}

// MessagesHandler allows adding a user to database
var MessagesHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := messagesResponse{}
	var uReq messagesRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing messages list request.")
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
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermListMessages); !ok {
			return
		}
	}
	messages, err := models.GetMessages(uReq.MessageCriteria)
	resp := routes.Response{}
	if err != nil {
		resp.Ok = false
		resp.Errors = routes.ResponseErrors{"db": "Couldn't get messages."}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.Messages = messages
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
