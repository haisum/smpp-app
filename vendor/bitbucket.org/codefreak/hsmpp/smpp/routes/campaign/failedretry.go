package campaign

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type retryRequest struct {
	CampaignID string
	URL        string
	Token      string
}

type retryResponse struct {
	Count int
}

//RetryHandler accepts post request to restart all messages with status error of a campaign
var RetryHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := stopResponse{}
	var uReq stopRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing retry request.")
		resp := routes.Response{}
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
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, user.PermRetryCampaign); !ok {
		return
	}
	msgs, err := models.GetErrorMessages(uReq.CampaignID)
	if err != nil {
		log.WithError(err).Error("Error getting error messages.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't restart campaign.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
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
	errCh := make(chan error, 1)
	okCh := make(chan bool, len(msgs))
	burstCh := make(chan int, 1000)
	for _, msg := range msgs {
		go func(m models.Message) {
			m.QueuedAt = time.Now().UTC().Unix()
			m.Status = models.MsgQueued
			errU := m.Update()
			if errU != nil {
				errCh <- errU
				return
			}
			noKey = group.DefaultPfx
			key := matchKey(keys, m.Dst, noKey)
			qItem := queue.Item{
				MsgID: m.ID,
				Total: m.Total,
			}
			respJSON, _ := qItem.ToJSON()
			errP := q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
			if errP != nil {
				errCh <- errP
				return
			}
			okCh <- true
			//free one burst
			<-burstCh
		}(msg)
		//proceed if you can feed the burst channel
		burstCh <- 1
	}
	for i := 0; i < len(msgs); i++ {
		select {
		case <-errCh:
			log.WithFields(log.Fields{
				"error": err,
				"uReq":  uReq,
			}).Error("Couldn't publish message.")
			resp := routes.Response{
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
		case <-okCh:
		}
	}
	log.Infof("%d campaign messages queued", len(msgs))
	uResp.Count = len(msgs)
	resp := routes.Response{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
