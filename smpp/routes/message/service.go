package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp/entities/message"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/smtext"
	"context"
	"io"
	"regexp"
	"strings"
	"time"
)

// Service is message service's interface
type Service interface {
	List(ctx context.Context, request listRequest) (listResponse, error)
	Send(ctx context.Context, request sendRequest) (sendResponse, error)
	ListDownload(ctx context.Context, request listDownloadRequest) (routes.AttachmentResponse, error)
	Stats(ctx context.Context, request statsRequest) (statsResponse, error)
}

type service struct {
	logger        logger.Logger
	msgStore      message.MessageStorer
	xlsExportFunc excelFunc
	authenticator user.Authenticator
}

type excelFunc func(m []message.Message, TZ string, cols []string) (func(writer io.Writer) (err error), error)

// NewService returns a new user service
func NewService(logger logger.Logger, msgStore message.MessageStorer, xlsExportFunc excelFunc, auth user.Authenticator) Service {
	return &service{
		logger, msgStore, xlsExportFunc, auth,
	}
}

// List endpoint returns info of user in current context
func (s *service) List(ctx context.Context, request listRequest) (listResponse, error) {
	response := listResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	if u.Username != request.Username && !u.Can(permission.ListUsers) {
		return response, errs.ForbiddenError{"user doesn't have list users permission"}
	}
	messages, err := s.msgStore.List(&request.Criteria)
	if err != nil {
		return response, err
	}
	response.Messages = messages
	return response, nil
}

// ListDownload endpoint returns io.Reader to download csv file generated for list
func (s *service) ListDownload(ctx context.Context, request listDownloadRequest) (routes.AttachmentResponse, error) {
	response := routes.AttachmentResponse{}
	listResp, err := s.List(ctx, request.listRequest)
	if err != nil {
		return response, err
	}
	writeFunc, err := s.xlsExportFunc(listResp.Messages, request.TZ, strings.Split(request.ReportCols, ","))
	response.Write = writeFunc
	response.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	response.Filename = "SMSReport.xlsx"
	return response, nil
}

// Stats endpoint returns stats of messages found against given criteria
// Stats contain number of messages in different statuses
func (s *service) Stats(ctx context.Context, request statsRequest) (statsResponse, error) {
	response := statsResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	if u.Username != request.Username && !u.Can(permission.ListUsers) {
		return response, errs.ForbiddenError{"user doesn't have list users permission"}
	}
	response.Stats, err = s.msgStore.Stats(&request.Criteria)
	return response, err
}

// Send endpoint stores given message in message store
func (s *service) Send(ctx context.Context, request sendRequest) (sendResponse, error) {
	response := sendResponse{}

	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}

	if request.Mask {
		if !u.Can(permission.Mask) {
			return response, &errs.ForbiddenError{Message: "user doesn't have masking permission"}
		}
	}
	errors := request.validate()
	if len(errors) > 0 {
		return response, errs.ErrorResponse{
			Errors: errors,
		}
	}

	var (
		queuedTime int64          = time.Now().UTC().Unix()
		status     message.Status = message.Queued
	)
	if request.ScheduledAt > 0 {
		status = message.Scheduled
	}
	enc := smtext.EncLatin
	if !smtext.IsASCII(request.Msg) {
		enc = smtext.EncUCS
	}
	m := &message.Message{
		ConnectionGroup: u.ConnectionGroup,
		Username:        u.Username,
		Msg:             request.Msg,
		Enc:             enc,
		Dst:             request.Dst,
		Src:             request.Src,
		Priority:        request.Priority,
		QueuedAt:        queuedTime,
		Status:          status,
		ScheduledAt:     request.ScheduledAt,
		SendAfter:       request.SendAfter,
		SendBefore:      request.SendBefore,
		IsFlash:         request.IsFlash,
	}
	msg := request.Msg
	if request.Mask {
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
	response.ID, err = s.msgStore.Save(m)
	return response, err
}
