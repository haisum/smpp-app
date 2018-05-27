package excel

import (
	"bitbucket.org/codefreak/hsmpp/pkg/entities/message"
	"github.com/tealeg/xlsx"
	"io"
	"strconv"
	"strings"
	"time"
)

var (
	labels = map[string]string{
		"Dst":     "Mobile Number",
		"Src":     "Sender ID",
		"Msg":     "Message",
		"IsFlash": "Flash Message",
	}
	availableCols = []string{
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
)

// ExportMessages exports given messages in a excel file. You can select timezone to export dates in.
// cols []string can be used to determine what columns should be included in exported excel file.
// returned function can be used to write file to any io.Writer
func ExportMessages(m []message.Message, TZ string, cols []string) (func(io.Writer) error, error) {
	if len(cols) == 0 || (len(cols) == 1 && cols[0] == "") {
		cols = availableCols
	} else {
		cols = trimUnknownColumns(cols, availableCols)
	}
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		return nil, err
	}

	addHeaders(sheet, cols)
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
	for _, v := range m {
		queued = formatTime(v.QueuedAt, loc)
		sent = formatTime(v.SentAt, loc)
		delivered = formatTime(v.DeliveredAt, loc)
		scheduled = formatTime(v.ScheduledAt, loc)
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
		addReportRow(sheet, cols, infoAvailable)
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

func formatTime(timestamp int64, location *time.Location) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).In(location).Format("02-01-2006 03:04:05 MST")
}

func trimUnknownColumns(cols, availableCols []string) []string {
	cols = trimSpace(cols)
	// trim all unknown columns
	for k, v := range cols {
		if !contains(availableCols, v) {
			cols = append(cols[:k], cols[k+1:]...)
		}
	}
	return cols
}

func addHeaders(sheet *xlsx.Sheet, cols []string) {
	row := sheet.AddRow()
	for _, v := range cols {
		cell := row.AddCell()
		if l, ok := labels[v]; ok {
			cell.Value = l
		} else {
			cell.Value = v
		}
	}
}

func addReportRow(sheet *xlsx.Sheet, cols []string, info map[string]string) {
	row := sheet.AddRow()
	for _, v := range cols {
		if val, ok := info[v]; ok {
			cell := row.AddCell()
			cell.Value = val
		}
	}
}
