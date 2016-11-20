package message

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/smtext"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type messageReq struct {
	Priority    int
	Src         string
	Dst         string
	Msg         string
	URL         string
	Token       string
	ScheduledAt int64
	IsFlash     bool
	SendBefore  string
	SendAfter   string
	Mask        bool
}

type messageResponse struct {
	ID string
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
	uReq.URL = r.URL.RequestURI()
	var (
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, user.PermSendMessage); !ok {
		return
	}
	if uReq.Mask {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, user.PermMask); !ok {
			return
		}
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
	var (
		queuedTime int64                = time.Now().UTC().Unix()
		status     models.MessageStatus = models.MsgQueued
	)
	if uReq.ScheduledAt > 0 {
		status = models.MsgScheduled
	}
	enc := smtext.EncLatin
	if !smtext.IsASCII(uReq.Msg) {
		enc = smtext.EncUCS
	}
	m := models.Message{
		ConnectionGroup: u.ConnectionGroup,
		Username:        u.Username,
		Msg:             uReq.Msg,
		Enc:             enc,
		Dst:             uReq.Dst,
		Src:             uReq.Src,
		Priority:        uReq.Priority,
		QueuedAt:        queuedTime,
		Status:          status,
		ScheduledAt:     uReq.ScheduledAt,
		SendAfter:       uReq.SendAfter,
		SendBefore:      uReq.SendBefore,
		IsFlash:         uReq.IsFlash,
	}
	msg := uReq.Msg
	if uReq.Mask {
		re := regexp.MustCompile("\\[\\[[^\\]]*\\]\\]")
		bs := re.FindAll([]byte(msg), -1)
		for i := 0; i < len(bs); i++ {
			val := strings.Trim(string(bs[i]), "[]")
			msg = strings.Replace(msg, "[["+val+"]]", val, -1)
			m.Msg = strings.Replace(m.Msg, "[["+val+"]]", strings.Repeat("X", len(val)), -1)
		}
	}
	m.RealMsg = msg
	m.Total = smtext.Total(msg, m.Enc)
	log.WithField("total", m.Total).Info("Total messages.")
	msgID, err := m.Save()
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
	if m.ScheduledAt == 0 {
		noKey = group.DefaultPfx
		key := matchKey(keys, uReq.Dst, noKey)
		qItem := queue.Item{
			MsgID: msgID,
			Total: m.Total,
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
	} else {
		log.WithField("ScheduledAt", time.Unix(m.ScheduledAt, 0).UTC().String()).Info("Scheduling message.")
	}
	uResp.ID = msgID
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)
})

func validateMsg(msg messageReq) []routes.ResponseError {
	var errors []routes.ResponseError
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
	if (msg.SendAfter == "" && msg.SendBefore != "") || (msg.SendBefore == "" && msg.SendAfter != "") {
		errors = append(errors, routes.ResponseError{
			Type:    routes.ErrorTypeRequest,
			Message: "Send before time and Send after time, both should be provided at a time.",
		})
	}
	parts := strings.Split(msg.SendAfter, ":")
	if msg.SendAfter != "" {
		if len(parts) != 2 {
			errors = append(errors, routes.ResponseError{
				Type:    routes.ErrorTypeForm,
				Field:   "SendAfter",
				Message: "Send after must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				errors = append(errors, routes.ResponseError{
					Type:    routes.ErrorTypeForm,
					Field:   "SendAfter",
					Message: "Send after must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	parts = strings.Split(msg.SendBefore, ":")
	if msg.SendBefore != "" {
		if len(parts) != 2 {
			errors = append(errors, routes.ResponseError{
				Type:    routes.ErrorTypeForm,
				Field:   "SendBefore",
				Message: "Send before must be of 24 hour format such as \"09:00\".",
			})
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				errors = append(errors, routes.ResponseError{
					Type:    routes.ErrorTypeForm,
					Field:   "SendBefore",
					Message: "Send before must be of 24 hour format such as \"09:00\".",
				})
			}
		}
	}
	if msg.ScheduledAt != 0 && msg.ScheduledAt < time.Now().UTC().Unix() {
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
