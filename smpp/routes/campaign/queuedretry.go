package campaign

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/message"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type retryQdRequest struct {
	CampaignID int64
	URL        string
	Token      string
}

type retryQdResponse struct {
	Count int
}

//RetryHandler accepts post request to restart all messages with status error of a campaign
var RetryQdHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := retryQdResponse{}
	var uReq retryQdRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing retry request.")
		resp := routes.ClientResponse{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeRequest,
				Message: "Couldn't parse request.",
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
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, permission.RetryCampaign); !ok {
		return
	}
	msgs, err := message.ListQueued(uReq.CampaignID)
	if err != nil {
		log.WithError(err).Error("Error getting queued messages.")
		resp := routes.ClientResponse{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't retry queued messages.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	q := queue.Get()
	config, err := smpp.GetConfig()
	keys := config.GetKeys(u.ConnectionGroup)
	var noKey string
	var group smpp.ConnGroup
	if group, err = config.GetGroup(u.ConnectionGroup); err != nil {
		log.WithField("ConnectionGroup", u.ConnectionGroup).Error("User's connection group doesn't exist in configuration.")
		resp := routes.ClientResponse{
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
	for _, m := range msgs {
		m.QueuedAt = time.Now().UTC().Unix()
		m.Status = message.Queued
		err = m.Update()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"uReq":  uReq,
			}).Error("Couldn't update messages.")
			resp := routes.ClientResponse{
				Errors: []routes.ResponseError{
					{
						Type:    routes.ErrorTypeQueue,
						Message: "Couldn't update messages.",
					},
				},
				Request: uReq,
			}
			resp.Send(w, *r, http.StatusInternalServerError)
			return
		}
		noKey = group.DefaultPfx
		key := matchKey(keys, m.Dst, noKey)
		qItem := queue.Item{
			MsgID: m.ID,
			Total: m.Total,
		}
		respJSON, _ := qItem.ToJSON()
		err = q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"uReq":  uReq,
			}).Error("Couldn't publish message.")
			resp := routes.ClientResponse{
				Errors: []routes.ResponseError{
					{
						Type:    routes.ErrorTypeQueue,
						Message: "Couldn't queue message.",
					},
				},
				Request: uReq,
			}
			resp.Send(w, *r, http.StatusInternalServerError)
			return
		}
	}
	log.Infof("%d campaign messages re-queued", len(msgs))
	uResp.Count = len(msgs)
	resp := routes.ClientResponse{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
