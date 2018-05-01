package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp/entities/message"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"context"
	"github.com/pkg/errors"
	"io"
	"strings"
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
}

type excelFunc func(m []message.Message, TZ string, cols []string) (func(writer io.Writer) (err error), error)

// NewService returns a new user service
func NewService(logger logger.Logger, msgStore message.MessageStorer, xlsExportFunc excelFunc) Service {
	return &service{
		logger, msgStore, xlsExportFunc,
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
		return response, errors.Wrap(errs.ForbiddenError{}, "user doesn't have list users permission")
	}
	messages, err := s.msgStore.List(request.Criteria)
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
// @todo
func (s *service) Stats(ctx context.Context, request statsRequest) (statsResponse, error) {
	response := statsResponse{}
	return response, nil
}

// Send endpoint sends a single message
// @todo
func (s *service) Send(ctx context.Context, request sendRequest) (sendResponse, error) {
	response := sendResponse{}

	if request.Mask {
		u, err := user.FromContext(ctx)
		if err != nil {
			return response, err
		}
		if !u.Can(permission.Mask) {
			return response, &errs.ForbiddenError{Message: "user doesn't have masking permission"}
		}
	}
	return response, nil
}
