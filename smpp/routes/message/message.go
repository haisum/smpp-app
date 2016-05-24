package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type messageReq struct {
	Enc      string
	Priority int
	Src      string
	Dst      string
	Msg      string
	Url      string
	Token    string
}

type messageResponse struct {
	Id string
}

// MessageHandler allows sending one sms
var MessageHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := messageResponse{}
	var uReq messageReq
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing user message request.")
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
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermSendMessage); !ok {
		return
	}
	if errors := validateMsg(uReq); len(errors) != 0 {
		log.WithField("errors", errors).Error("Validation failed.")
		resp := routes.Response{
			Errors:  errors,
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	q, err := queue.GetQueue("", "", 0)
	config, err := models.GetConfig()
	keys := config.GetKeys(u.ConnectionGroup)
	var noKey string
	var group smpp.ConnGroup
	if group, err = config.GetGroup(u.ConnectionGroup); err != nil {
		log.WithField("ConnectionGroup", u.ConnectionGroup).Error("User's connection group doesn't exist in configuration.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "User's connection group doesn't exist in configuration.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}

	m := models.Message{
		ConnectionGroup: u.ConnectionGroup,
		Username:        u.Username,
		Msg:             uReq.Msg,
		Enc:             uReq.Enc,
		Dst:             uReq.Dst,
		Src:             uReq.Src,
		Priority:        uReq.Priority,
		QueuedAt:        time.Now().Unix(),
		Status:          models.MsgSubmitted,
	}
	msgId, err := m.Save()
	if err != nil {
		log.WithField("err", err).Error("Couldn't insert in db.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeDB,
					Message: "Couldn't save message in database.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	noKey = group.DefaultPfx
	key := matchKey(keys, uReq.Dst, noKey)
	qItem := queue.Item{
		Priority: uReq.Priority,
		Enc:      uReq.Enc,
		Src:      uReq.Src,
		Msg:      uReq.Msg,
		Dst:      uReq.Dst,
		MsgId:    msgId,
	}
	respJSON, _ := qItem.ToJSON()
	err = q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(uReq.Priority))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"uReq":  uReq,
		}).Error("Couldn't publish message.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeQueue,
					Message: "Couldn't send message.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)
})

func validateMsg(msg messageReq) []routes.ResponseError {
	errors := make([]routes.ResponseError, 0)
	if msg.Dst == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Dst",
			Message: "Destination can't be empty.",
		})
	}
	if msg.Msg == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Msg",
			Message: "Can't send empty message",
		})
	}
	if msg.Src == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Src",
			Message: "Source address can't be empty.",
		})
	}
	if msg.Enc != "ucs" && msg.Enc != "latin" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Enc",
			Message: "Encoding can either be latin or UCS",
		})
	}
	return errors
}

// Given a list of strings and a string,
// this function returns a list item if large string starts with list item.
// string in parameter noKey is returned if no matches could be found
func matchKey(keys []string, str string, noKey string) string {
	for _, key := range keys {
		if strings.HasPrefix(str, key) {
			return key
		}
	}
	return noKey
}
