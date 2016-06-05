package message

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type messagesRequest struct {
	models.MessageCriteria
	URL   string
	Token string
	CSV   bool
	Stats bool
	TZ    string
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
	uReq.URL = r.URL.RequestURI()
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
	if uReq.Stats == true {
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
		uResp.Stats = stats
	}
	if uReq.CSV == true {
		toCsv(w, r, messages, uReq.TZ)
	} else {
		uResp.Messages = messages
		resp.Obj = uResp
		resp.Ok = true
		resp.Request = uReq
		resp.Send(w, *r, http.StatusOK)
	}
})

func toCsv(w http.ResponseWriter, r *http.Request, m []models.Message, TZ string) {
	b := &bytes.Buffer{}
	wr := csv.NewWriter(b)
	wr.Write([]string{
		"ID",
		"Connection",
		"ConnectionGroup",
		"Status",
		"Error",
		"RespID",
		"Total",
		"Username",
		"Msg",
		"Enc",
		"Dst",
		"Src",
		"CampaignID",
		"Priority",
		"QueuedAt",
		"SentAt",
		"DeliveredAt",
		"ScheduledAt",
		"SendBefore",
		"SendAfter",
	})
	for _, v := range m {
		var (
			queued    string
			sent      string
			delivered string
			scheduled string
		)
		loc, err := time.LoadLocation(TZ)
		if err != nil {
			log.WithFields(log.Fields{"Error": err, "TZ": TZ}).Error("Couldn't load location. Loading UTC")
			loc, _ = time.LoadLocation("UTC")
		}
		if v.QueuedAt > 0 {
			queued = time.Unix(v.QueuedAt, 0).In(loc).Format("02-01-2006 03:04:05 MST")
		}
		if v.SentAt > 0 {
			sent = time.Unix(v.SentAt, 0).In(loc).Format("02-01-2006 03:04:05 MST")
		}
		if v.DeliveredAt > 0 {
			delivered = time.Unix(v.DeliveredAt, 0).In(loc).Format("02-01-2006 03:04:05 MST")
		}
		if v.ScheduledAt > 0 {
			scheduled = time.Unix(v.ScheduledAt, 0).In(loc).Format("02-01-2006 03:04:05 MST")
		}
		wr.Write([]string{
			v.ID,
			v.Connection,
			v.ConnectionGroup,
			string(v.Status),
			v.Error,
			v.RespID,
			strconv.Itoa(v.Total),
			v.Username,
			v.Msg,
			v.Enc,
			v.Dst,
			v.Src,
			v.CampaignID,
			strconv.Itoa(v.Priority),
			queued,
			sent,
			delivered,
			scheduled,
			v.SendBefore,
			v.SendAfter,
		})
	}
	wr.Flush()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=SMSReport.csv")
	w.Write(b.Bytes())
}
