package campaign

import (
	"context"

	"regexp"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/pkg/entities/campaign"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/campaign/file"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/message"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/user"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/errs"
	"bitbucket.org/codefreak/hsmpp/pkg/logger"
	"bitbucket.org/codefreak/hsmpp/pkg/stringutils"
	"github.com/pkg/errors"
)

// Service is interface for campaign service
type Service interface {
	List(ctx context.Context, request listRequest) (listResponse, error)
	Start(ctx context.Context, request startRequest) (startResponse, error)
	Progress(ctx context.Context, request progressRequest) (progressResponse, error)
	Stop(ctx context.Context, request stopRequest) (stopResponse, error)
	Report(ctx context.Context, request reportRequest) (reportResponse, error)
}

type service struct {
	logger           logger.Logger
	campaignStore    campaign.Store
	messageStore     message.Store
	fileStore        file.Store
	processExcelFunc file.ProcessExcelFunc
	fileManager      file.OpenReadWriteCloser
	authenticator    user.Authenticator
}

// NewService returns a new user service
func NewService(logger logger.Logger, campaignStore campaign.Store, messageStore message.Store, fileStore file.Store, fileManager file.OpenReadWriteCloser, processExcelFunc file.ProcessExcelFunc, auth user.Authenticator) Service {
	return &service{
		logger, campaignStore, messageStore,
		fileStore, processExcelFunc, fileManager,
		auth,
	}
}

// List filters campaigns according to criteria given in request
func (svc *service) List(ctx context.Context, request listRequest) (listResponse, error) {
	response := listResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	if request.Username != u.Username && !u.Can(permission.ListCampaigns) {
		return response, errs.ForbiddenError{"user doesn't have list campaign permission"}
	}
	response.Campaigns, err = svc.campaignStore.List(&request.Criteria)
	return response, err
}

func (svc *service) Start(ctx context.Context, request startRequest) (startResponse, error) {
	response := startResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	if request.Mask {
		if !u.Can(permission.Mask) {
			return response, errs.ForbiddenError{"user doesn't have mask permissions"}
		}
	}
	var numbers []file.Row
	if request.FileID == 0 {
		if request.Numbers != "" {
			numbers = file.RowsFromString(request.Numbers)
		} else {
			resp := errs.ErrorResponse{}
			resp.Errors = []errs.ResponseError{
				{
					Type:    errs.ErrorTypeRequest,
					Message: "No numbers provided. You should either select a file or send comma separated list of numbers",
				},
			}
			return response, errors.Wrap(resp, err.Error())
		}
	} else {
		var files []file.File
		files, err := svc.fileStore.List(&file.Criteria{
			ID: request.FileID,
		})
		if err != nil || len(files) == 0 {
			resp := errs.ErrorResponse{}
			resp.Errors = []errs.ResponseError{
				{
					Type:    errs.ErrorTypeForm,
					Message: "Couldn't get any file.",
					Field:   "FileID",
				},
			}
			return response, errors.Wrap(resp, err.Error())
		}
		reader, err := svc.fileManager.Open(files[0].Name)
		if err != nil {
			return response, err
		}
		defer reader.Close()
		numbers, err = file.ToNumbers(&files[0], svc.processExcelFunc, reader)
		if err != nil {
			resp := errs.ErrorResponse{}
			resp.Errors = []errs.ResponseError{
				{
					Type:    errs.ErrorTypeForm,
					Message: "Couldn't read numbers from file.",
					Field:   "FileID",
				},
			}
			return response, errors.Wrap(resp, err.Error())
		}
	}

	c := campaign.Campaign{
		Description: request.Description,
		Src:         request.Src,
		Msg:         request.Msg,
		FileID:      request.FileID,
		SubmittedAt: time.Now().UTC().Unix(),
		Priority:    request.Priority,
		SendBefore:  request.SendBefore,
		SendAfter:   request.SendAfter,
		ScheduledAt: request.ScheduledAt,
		Username:    u.Username,
	}

	if errors := request.validate(); len(errors) != 0 {
		respErr := errs.ErrorResponse{
			Errors: errors,
		}
		return response, respErr
	}
	msg := request.Msg
	if request.Mask {
		re := regexp.MustCompile("\\[\\[[^\\]]*\\]\\]")
		bs := re.FindAll([]byte(msg), -1)
		for i := 0; i < len(bs); i++ {
			val := strings.Trim(string(bs[i]), "[]")
			msg = strings.Replace(msg, "[["+val+"]]", val, -1)
			c.Msg = strings.Replace(c.Msg, "[["+val+"]]", strings.Repeat("X", len(val)), -1)
		}
	}
	c.Total = len(numbers)
	c.ID, err = svc.campaignStore.Save(&c)
	if err != nil {
		respErr := errs.ErrorResponse{
			Errors: []errs.ResponseError{
				{
					Type:    errs.ErrorTypeDB,
					Message: "Couldn't save campaign in db.",
				},
			},
		}
		return response, respErr
	}
	go svc.saveMessages(request, &c, u, numbers, msg)
	response.ID = c.ID
	return response, nil
}

func (svc *service) saveMessages(request startRequest, c *campaign.Campaign, u *user.User, numbers []file.Row, msg string) {
	enc := message.EncLatin
	if len(numbers) > 0 {
		encMsg := msg
		for search, replace := range numbers[0].Params {
			encMsg = strings.Replace(encMsg, "{{"+search+"}}", replace, -1)
		}
		if !stringutils.IsASCII(encMsg) {
			enc = message.EncUCS
		}
	}
	total := message.Total(msg, enc)

	var ms []message.Message
	c.Errors = make([]string, 0)
	for i, nr := range numbers {
		var (
			queuedTime = time.Now().UTC().Unix()
			status     = message.Queued
		)
		if request.ScheduledAt > 0 {
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
			realTotal = message.Total(realMsg, enc)
		}
		m := message.Message{
			ConnectionGroup: u.ConnectionGroup,
			Username:        u.Username,
			Msg:             maskedMsg,
			RealMsg:         realMsg,
			Enc:             enc,
			Dst:             nr.Destination,
			Src:             request.Src,
			Priority:        request.Priority,
			QueuedAt:        queuedTime,
			Status:          status,
			CampaignID:      c.ID,
			SendBefore:      request.SendBefore,
			SendAfter:       request.SendAfter,
			ScheduledAt:     request.ScheduledAt,
			Total:           realTotal,
			Campaign:        request.Description,
			IsFlash:         request.IsFlash,
		}
		ms = append(ms, m)
		// if we have MaxInsertCount messages or last few messages
		if (i+1)%svc.messageStore.MaxInsertCount() == 0 || (i+1) == len(numbers) {
			_, err := svc.messageStore.SaveBulk(ms)
			if err != nil {
				c.Errors = append(c.Errors, err.Error())
			}
			ms = []message.Message{}
		}
	}
	if len(c.Errors) > 0 {
		svc.campaignStore.Save(c)
	}
}

// Progress returns count of messages in different status in given campaign
func (svc *service) Progress(ctx context.Context, request progressRequest) (progressResponse, error) {
	cp, err := svc.campaignStore.List(&campaign.Criteria{ID: request.CampaignID})
	var (
		p        campaign.Progress
		response progressResponse
	)
	if err == nil && len(cp) > 0 {
		p, err = svc.campaignStore.Progress(cp[0].ID)
	} else {
		respErr := errs.ErrorResponse{}
		respErr.Errors = []errs.ResponseError{
			{
				Type:    errs.ErrorTypeDB,
				Message: "couldn't get campaign progress",
			},
		}
		return response, respErr
	}
	response.Progress = p
	return response, nil
}

func (svc *service) Stop(ctx context.Context, request stopRequest) (stopResponse, error) {
	response := stopResponse{}
	count, err := svc.messageStore.StopPending(request.CampaignID)
	if err != nil {
		respErr := errs.ErrorResponse{}
		respErr.Errors = []errs.ResponseError{
			{
				Type:    errs.ErrorTypeDB,
				Message: "Couldn't update campaign.",
			},
		}

		return response, respErr
	}
	response.Count = count
	return response, nil
}

func (svc *service) Report(ctx context.Context, request reportRequest) (reportResponse, error) {
	response := reportResponse{}
	c, err := svc.campaignStore.List(&campaign.Criteria{ID: request.CampaignID})
	var cr campaign.Report
	if len(c) > 0 && err == nil {
		cr, err = svc.campaignStore.Report(c[0].ID)
	}
	if err != nil {
		respErr := errs.ErrorResponse{}
		respErr.Errors = []errs.ResponseError{
			{
				Type:    errs.ErrorTypeDB,
				Message: "Couldn't get campaign report.",
			},
		}
		return response, respErr
	}
	response.Report = cr
	return response, nil
}
