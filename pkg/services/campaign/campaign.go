package campaign

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/pkg"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign/file"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/message"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/queue"
)

const (
	// maxBulkInsert is number of msgs to insert at a time.
	maxBulkInsert = 200
)

// Handler allows starting a campaign
var Handlerx = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := campaignResponse{}
	var uReq campaignRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign request.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeRequest,
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
	if u, ok = services.Authenticate(w, *r, uReq, uReq.Token, permission.StartCampaign); !ok {
		return
	}
	if uReq.Mask {
		if _, ok = services.Authenticate(w, *r, uReq, uReq.Token, permission.Mask); !ok {
			return
		}
	}
	var numbers []file.Row
	if uReq.FileID != 0 {
		var files []file.NumFile
		files, err = file.List(file.Criteria{
			ID: uReq.FileID,
		})
		if err != nil || len(files) == 0 {
			resp := services.ClientResponse{}
			resp.Errors = []services.ResponseError{
				{
					Type:    services.ErrorTypeForm,
					Message: "Couldn't get any file.",
					Field:   "FileID",
				},
			}
			resp.Send(w, *r, http.StatusBadRequest)
		}
		numbers, err = files[0].ToNumbers(&file.RealNumFileIO{})
		if err != nil {
			log.WithError(err).Error("Couldn't read numbers from file.")
			resp := services.ClientResponse{}
			resp.Errors = []services.ResponseError{
				{
					Type:    services.ErrorTypeForm,
					Message: "Couldn't read numbers from file.",
					Field:   "FileID",
				},
			}
			resp.Send(w, *r, http.StatusInternalServerError)
		}
	} else if uReq.Numbers != "" {
		numbers = file.RowsFromString(uReq.Numbers)
	} else {
		log.WithError(err).Error("No numbers provided.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeRequest,
				Message: "No numbers provided. You should either select a file or send comma separated list of numbers",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
	}
	c := campaign.Campaign{
		Description: uReq.Description,
		Src:         uReq.Src,
		Msg:         uReq.Msg,
		FileID:      uReq.FileID,
		SubmittedAt: time.Now().UTC().Unix(),
		Priority:    uReq.Priority,
		SendBefore:  uReq.SendBefore,
		SendAfter:   uReq.SendAfter,
		ScheduledAt: uReq.ScheduledAt,
		Username:    u.Username,
	}

	if errors := validateCampaign(uReq); len(errors) != 0 {
		log.WithField("errors", errors).Error("Validation failed.")
		resp := services.ClientResponse{
			Errors:  errors,
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	msg := uReq.Msg
	if uReq.Mask {
		re := regexp.MustCompile("\\[\\[[^\\]]*\\]\\]")
		bs := re.FindAll([]byte(msg), -1)
		for i := 0; i < len(bs); i++ {
			val := strings.Trim(string(bs[i]), "[]")
			msg = strings.Replace(msg, "[["+val+"]]", val, -1)
			c.Msg = strings.Replace(c.Msg, "[["+val+"]]", strings.Repeat("X", len(val)), -1)
		}
	}
	c.Total = len(numbers)
	campaignID, err := c.Save()
	if err != nil {
		log.WithError(err).Error("Couldn't save campaign.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeDB,
					Message: "Couldn't save campaign in db.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
	}
	q := queue.Get()
	config, err := pkg.GetConfig()
	if err != nil {
		log.WithError(err).Error("Couldn't Get config.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeDB,
					Message: "Couldn't get config.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
	}
	keys := config.GetKeys(u.ConnectionGroup)
	var group pkg.ConnGroup
	if group, err = config.GetGroup(u.ConnectionGroup); err != nil {
		log.WithField("ConnectionGroup", u.ConnectionGroup).Error("User's connection group doesn't exist in configuration.")
		resp := services.ClientResponse{
			Errors: []services.ResponseError{
				{
					Type:    services.ErrorTypeConfig,
					Message: "User's connection group doesn't exist in configuration.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	noKey := group.DefaultPfx
	enc := smtext.EncLatin
	if len(numbers) > 0 {
		encMsg := msg
		for search, replace := range numbers[0].Params {
			encMsg = strings.Replace(encMsg, "{{"+search+"}}", replace, -1)
		}
		if !smtext.IsASCII(encMsg) {
			enc = smtext.EncUCS
		}
	}
	total := smtext.Total(msg, enc)

	log.Info("Campaign messages are being queued.")
	uResp.ID = campaignID
	resp := services.ClientResponse{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)

	go func() {
		var ms []message.Message
		c.Errors = make([]string, 0)
		for i, nr := range numbers {
			var (
				queuedTime int64          = time.Now().UTC().Unix()
				status     message.Status = message.Queued
			)
			if uReq.ScheduledAt > 0 {
				status = message.Scheduled
			}
			maskedMsg := c.Msg
			realMsg := msg
			for search, replace := range nr.Params {
				realMsg = strings.Replace(realMsg, "{{"+search+"}}", replace, -1)
				maskedMsg = strings.Replace(maskedMsg, "{{"+search+"}}", replace, -1)
			}
			realTotal := total
			if msg != realMsg {
				realTotal = smtext.Total(realMsg, enc)
			}
			m := message.Message{
				ConnectionGroup: u.ConnectionGroup,
				Username:        u.Username,
				Msg:             maskedMsg,
				RealMsg:         realMsg,
				Enc:             enc,
				Dst:             nr.Destination,
				Src:             uReq.Src,
				Priority:        uReq.Priority,
				QueuedAt:        queuedTime,
				Status:          status,
				CampaignID:      campaignID,
				SendBefore:      uReq.SendBefore,
				SendAfter:       uReq.SendAfter,
				ScheduledAt:     uReq.ScheduledAt,
				Total:           realTotal,
				Campaign:        uReq.Description,
				IsFlash:         uReq.IsFlash,
			}
			ms = append(ms, m)
			// if we have 200 msgs or last few messages
			if (i+1)%maxBulkInsert == 0 || (i+1) == len(numbers) {
				ids, err := message.SaveBulk(ms)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
						"uReq":  uReq,
					}).Error("Couldn't save messages.")
					c.Errors = append(c.Errors, err.Error())
				}
				for j, m := range ms {
					if m.ScheduledAt == 0 {
						key := matchKey(keys, m.Dst, noKey)
						qItem := queue.Item{
							MsgID: ids[j], // m.ID is empty.
							Total: m.Total,
						}
						respJSON, _ := qItem.ToJSON()
						err = q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(uReq.Priority))
						if err != nil {
							log.WithFields(log.Fields{
								"error": err,
								"uReq":  uReq,
							}).Error("Couldn't publish message.")
							c.Errors = append(c.Errors, err.Error())
						}
					}
				}
				ms = []message.Message{}
			}
		}
		if len(c.Errors) > 0 {
			c.Save()
		}
	}()

})

func validateCampaign(c campaignRequest) []services.ResponseError {
	var errors []services.ResponseError
	if c.Msg == "" {
		errors = append(errors, services.ResponseError{
			Type:    services.ErrorTypeForm,
			Field:   "Msg",
			Message: "Can't send empty message.",
		})
	}
	if c.Description == "" {
		errors = append(errors, services.ResponseError{
			Type:    services.ErrorTypeForm,
			Field:   "Description",
			Message: "Description must be provided for campaign.",
		})
	}
	if c.Src == "" {
		errors = append(errors, services.ResponseError{
			Type:    services.ErrorTypeForm,
			Field:   "Src",
			Message: "Source address can't be empty.",
		})
	}
	if (c.SendAfter == "" && c.SendBefore != "") || (c.SendBefore == "" && c.SendAfter != "") {
		errors = append(errors, services.ResponseError{
			Type:    services.ErrorTypeRequest,
			Message: "Send before time and Send after time, both should be provided at a time.",
		})
	}
	if c.SendAfter != "" {
		parts := strings.Split(c.SendAfter, ":")
		if len(parts) != 2 {
			errors = append(errors, services.ResponseError{
				Type:    services.ErrorTypeForm,
				Field:   "SendAfter",
				Message: "Send after must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 {
				errors = append(errors, services.ResponseError{
					Type:    services.ErrorTypeForm,
					Field:   "SendAfter",
					Message: "Send after must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	if c.SendBefore != "" {
		parts := strings.Split(c.SendBefore, ":")
		if len(parts) != 2 {
			errors = append(errors, services.ResponseError{
				Type:    services.ErrorTypeForm,
				Field:   "SendBefore",
				Message: "Send before must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 {
				errors = append(errors, services.ResponseError{
					Type:    services.ErrorTypeForm,
					Field:   "SendBefore",
					Message: "Send before must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	if c.ScheduledAt != 0 && c.ScheduledAt < time.Now().UTC().Unix() {
		errors = append(errors, services.ResponseError{
			Type:    services.ErrorTypeForm,
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