package campaign

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
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
	ScheduledAt int64
	SendBefore  string
	SendAfter   string
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
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeForm,
				Message: "Couldn't get any file.",
				Field:   "FileId",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
	}
	numbers, err := files[0].ToNumbers()
	if err != nil {
		log.WithError(err).Error("Couldn't read numbers from file.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeForm,
				Message: "Couldn't read numbers from file.",
				Field:   "FileId",
			},
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
		SendBefore:  uReq.SendBefore,
		SendAfter:   uReq.SendAfter,
		ScheduledAt: uReq.ScheduledAt,
	}

	if errors := validateCampaign(uReq); len(errors) != 0 {
		log.WithField("errors", errors).Error("Validation failed.")
		resp := routes.Response{
			Errors:  errors,
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	campaignId, err := c.Save()
	if err != nil {
		log.WithError(err).Error("Couldn't save campaign.")
		resp := routes.Response{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeDB,
					Message: "Couldn't save campaign in db.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
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
	okCh := make(chan bool, len(numbers))
	burstCh := make(chan int, 1000)
	for _, dst := range numbers {
		go func(dst string) {
			m := models.Message{
				ConnectionGroup: u.ConnectionGroup,
				Username:        u.Username,
				Msg:             uReq.Msg,
				Enc:             uReq.Enc,
				Dst:             dst,
				Src:             uReq.Src,
				Priority:        uReq.Priority,
				QueuedAt:        time.Now().Unix(),
				Status:          models.MsgQueued,
				CampaignId:      campaignId,
				SendBefore:      uReq.SendBefore,
				SendAfter:       uReq.SendAfter,
				ScheduledAt:     uReq.ScheduledAt,
			}
			msgId, err := m.Save()
			if err != nil {
				errCh <- err
				return
			}
			noKey = group.DefaultPfx
			key := matchKey(keys, dst, noKey)
			qItem := queue.Item{
				MsgId: msgId,
			}
			respJSON, _ := qItem.ToJSON()
			err = q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(uReq.Priority))
			if err != nil {
				errCh <- err
			} else {
				okCh <- true
			}
			//free one burst
			<-burstCh
		}(dst)
		//proceed if you can feed the burst channel
		burstCh <- 1
	}
	for i := 0; i < len(numbers); i++ {
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
	log.Info("All campaign messages queued")
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)

})

func validateCampaign(c campaignRequest) []routes.ResponseError {
	errors := make([]routes.ResponseError, 0)
	if c.FileId == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "FileId",
			Message: "File can't be empty.",
		})
	}
	if c.Msg == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Msg",
			Message: "Can't send empty message.",
		})
	}
	if c.Description == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Description",
			Message: "Description must be provided for campaign.",
		})
	}
	if c.Src == "" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Src",
			Message: "Source address can't be empty.",
		})
	}
	if c.Enc != "ucs" && c.Enc != "latin" {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "Enc",
			Message: "Encoding can either be latin or UCS.",
		})
	}
	if (c.SendAfter == "" && c.SendBefore != "") || (c.SendBefore == "" && c.SendAfter != "") {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeRequest,
			Message: "Send before time and Send after time, both should be provided at a time.",
		})
	}
	re := regexp.MustCompile("[0-9][0-9]:[0-9][0-9](AM)|(PM)")
	if c.SendAfter != "" && !re.Match([]byte(c.SendAfter)) {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "SendAfter",
			Message: "Send after must be of 24 hour format such as \"09:00\".",
		})
	}
	if c.SendBefore != "" && !re.Match([]byte(c.SendBefore)) {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "SendBefore",
			Message: "Send before must be of 24 hour format such as \"22:00\".",
		})
	}
	if c.ScheduledAt != 0 && c.ScheduledAt < time.Now().UTC().Unix() {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeForm,
			Field:   "ScheduledAt",
			Message: "Schedule time must be in future.",
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
