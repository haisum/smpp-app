package campaign

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

type campaignRequest struct {
	Url         string
	Token       string
	FileId      string
	Description string
	Priority    int
	Src         string
	Msg         string
	Enc         string
}

type campaignResponse struct {
	Id string
}

// CampaignHandler allows starting a campaign
var CampaignHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := campaignResponse{}
	var uReq campaignRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign request.")
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
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermStartCampaign); !ok {
		return
	}
	files, err := models.GetNumFiles(models.NumFileCriteria{
		Id: uReq.FileId,
	})
	if err != nil || len(files) == 0 {
		resp := routes.Response{
			Errors:  routes.ResponseErrors{"FileId": "Couldn't get any file."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
	}
	numbers, err := files[0].ToNumbers()
	if err != nil {
		log.WithError(err).Error("Couldn't read numbers from file.")
		resp := routes.Response{
			Errors:  routes.ResponseErrors{"FileId": "Couldn't read numbers from file."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
	}
	c := models.Campaign{
		Description: uReq.Description,
		Enc:         uReq.Enc,
		Src:         uReq.Src,
		Msg:         uReq.Msg,
		FileId:      uReq.FileId,
		SubmittedAt: time.Now().Unix(),
		Priority:    uReq.Priority,
	}
	campaignId, err := c.Save()
	if err != nil {
		log.WithError(err).Error("Couldn't save campaign.")
		resp := routes.Response{
			Errors:  routes.ResponseErrors{"db": "Couldn't save campaign in db."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
	}

	errors := make(routes.ResponseErrors)
	if errors = validateCampaign(uReq); len(errors) != 0 {
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
			Errors:  routes.ResponseErrors{"ConnectionGroup": "User's connection group doesn't exist in configuration."},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}

	for _, dst := range numbers {

		m := models.Message{
			ConnectionGroup: u.ConnectionGroup,
			Username:        u.Username,
			Msg:             uReq.Msg,
			Enc:             uReq.Enc,
			Dst:             dst,
			Src:             uReq.Src,
			Priority:        uReq.Priority,
			QueuedAt:        time.Now().Unix(),
			Status:          models.MsgSubmitted,
			CampaignId:      campaignId,
		}
		msgId, err := m.Save()
		if err != nil {
			log.WithField("err", err).Error("Couldn't insert in db.")
			resp := routes.Response{
				Errors:  routes.ResponseErrors{"db": "Couldn't save message in database."},
				Request: uReq,
			}
			resp.Send(w, *r, http.StatusInternalServerError)
			return
		}
		noKey = group.DefaultPfx
		key := matchKey(keys, dst, noKey)
		qItem := queue.Item{
			Priority: uReq.Priority,
			Enc:      uReq.Enc,
			Src:      uReq.Src,
			Msg:      uReq.Msg,
			Dst:      dst,
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
				Errors:  routes.ResponseErrors{"message": "Couldn't send message."},
				Request: uReq,
			}
			resp.Send(w, *r, http.StatusInternalServerError)
			return
		}
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)

})

func validateCampaign(c campaignRequest) routes.ResponseErrors {
	errors := make(routes.ResponseErrors)
	if c.FileId == "" {
		errors["FileId"] = "File can't be empty."
	}
	if c.Msg == "" {
		errors["Msg"] = "Can't send empty message"
	}
	if c.Src == "" {
		errors["Src"] = "Source address can't be empty."
	}
	if c.Enc != "ucs" && c.Enc != "latin" {
		errors["Enc"] = "Encoding can either be latin or UCS"
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
