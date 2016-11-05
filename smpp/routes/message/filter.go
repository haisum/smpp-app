package message

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
	"github.com/tealeg/xlsx"
)

type messagesRequest struct {
	models.MessageCriteria
	URL   string
	Token string
	XLSX  bool
	//commma separated list of columns to populate
	ReportCols string
	Stats      bool
	TZ         string
}

var (
	labels = map[string]string{
		"Dst":     "Mobile Number",
		"Src":     "Sender ID",
		"Msg":     "Message",
		"IsFlash": "Flash Message",
	}
)

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
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, user.PermListMessages); !ok {
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
	if uReq.XLSX == true {
		toXLS(w, r, messages, uReq.TZ, strings.Split(uReq.ReportCols, ","))
	} else {
		uResp.Messages = messages
		resp.Obj = uResp
		resp.Ok = true
		resp.Request = uReq
		resp.Send(w, *r, http.StatusOK)
	}
})

func toXLS(w http.ResponseWriter, r *http.Request, m []models.Message, TZ string, cols []string) {
	availableCols := []string{
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
		"Campaign",
		"Priority",
		"QueuedAt",
		"SentAt",
		"DeliveredAt",
		"ScheduledAt",
		"SendBefore",
		"SendAfter",
		"isFlash",
	}
	if len(cols) == 0 || (len(cols) == 1 && cols[0] == "") {
		cols = availableCols
	} else {
		cols = trimSpace(cols)
		//trim all unknown columns
		for k, v := range cols {
			if !contains(availableCols, v) {
				cols = append(cols[:k], cols[k+1:]...)
			}
		}
	}
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		fmt.Printf(err.Error())
	}

	row := sheet.AddRow()
	for _, v := range cols {
		cell := row.AddCell()
		if l, ok := labels[v]; ok {
			cell.Value = l
		} else {
			cell.Value = v
		}
	}
	for _, v := range m {
		var (
			queued    string
			sent      string
			delivered string
			scheduled string
			loc       *time.Location
		)
		loc, err = time.LoadLocation(TZ)
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
		infoAvailable := map[string]string{
			"ID":              v.ID,
			"Connection":      v.Connection,
			"ConnectionGroup": v.ConnectionGroup,
			"Status":          string(v.Status),
			"Error":           v.Error,
			"RespID":          v.RespID,
			"Total":           strconv.Itoa(v.Total),
			"Username":        v.Username,
			"Msg":             v.Msg,
			"Enc":             v.Enc,
			"Dst":             v.Dst,
			"Src":             v.Src,
			"CampaignID":      v.CampaignID,
			"Campaign":        v.Campaign,
			"Priority":        strconv.Itoa(v.Priority),
			"QueuedAt":        queued,
			"SentAt":          sent,
			"DeliveredAt":     delivered,
			"ScheduledAt":     scheduled,
			"SendBefore":      v.SendBefore,
			"SendAfter":       v.SendAfter,
			"IsFlash":         strconv.FormatBool(v.IsFlash),
		}
		row = sheet.AddRow()
		for _, v := range cols {
			if val, ok := infoAvailable[v]; ok {
				cell := row.AddCell()
				cell.Value = val
			}
		}
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment;filename=SMSReport.xlsx")
	err = file.Write(w)
	if err != nil {
		log.WithError(err).Error("Excel file writing failed.")
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func trimSpace(s []string) []string {
	var trS []string
	for _, v := range s {
		trS = append(trS, strings.TrimSpace(v))
	}
	return trS
}
