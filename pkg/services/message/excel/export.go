package excel

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/pkg/entities/message"
	"github.com/tealeg/xlsx"
)

var (
	labels = map[string]string{
		"Dst":     "Mobile Number",
		"Src":     "Sender ID",
		"Msg":     "Message",
		"IsFlash": "Flash Message",
	}
)

// ExportMessages exports given messages in a excel file. You can select timezone to export dates in.
// cols []string can be used to determine what columns should be included in exported excel file.
// returned function can be used to write file to any io.Writer
func ExportMessages(m []message.Message, TZ string, cols []string) (func(io.Writer) error, error) {
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
		"IsFlash",
	}
	if len(cols) == 0 || (len(cols) == 1 && cols[0] == "") {
		cols = availableCols
	} else {
		cols = trimSpace(cols)
		// trim all unknown columns
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
			"ID":              strconv.FormatInt(v.ID, 10),
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
			"CampaignID":      strconv.FormatInt(v.CampaignID, 10),
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
	return file.Write, err
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
