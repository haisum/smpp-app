package file

import (
	"context"

	"path/filepath"

	"time"

	"github.com/haisum/smpp-app/pkg/entities/campaign/file"
	"github.com/haisum/smpp-app/pkg/entities/user"
	"github.com/haisum/smpp-app/pkg/entities/user/permission"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/haisum/smpp-app/pkg/logger"
	"github.com/haisum/smpp-app/pkg/response"
	"github.com/pkg/errors"
)

// Service is interface for campaign service
type Service interface {
	Delete(ctx context.Context, request deleteRequest) (deleteResponse, error)
	Download(ctx context.Context, request downloadRequest) (response.Attachment, error)
	List(ctx context.Context, request listRequest) (listResponse, error)
	Upload(ctx context.Context, request uploadRequest) (uploadResponse, error)
}

type service struct {
	logger           logger.Logger
	fileStore        file.Store
	fileManager      file.OpenReadWriteCloser
	processExcelFunc file.ProcessExcelFunc
	randFunc         func() string
	authenticator    user.Authenticator
}

// NewService returns a new user service
func NewService(logger logger.Logger, fileStore file.Store, fileManager file.OpenReadWriteCloser, processExcelFunc file.ProcessExcelFunc, randFunc func() string, auth user.Authenticator) Service {
	return &service{
		logger,
		fileStore, fileManager,
		processExcelFunc, randFunc,
		auth,
	}
}

// Delete marks a file as deleted
func (svc *service) Delete(ctx context.Context, request deleteRequest) (deleteResponse, error) {
	response := deleteResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	files, err := svc.fileStore.List(&file.Criteria{
		ID: request.ID,
	})
	if len(files) == 0 {
		svc.logger.Error("msg", err)
		return response, errors.New("couldn't get file")
	} else if files[0].Username != u.Username {
		if ok := u.Can(permission.DeleteCampaignFile); !ok {
			return response, errs.ForbiddenError{"user doesn't have permission to delete campaign file"}
		}
	}
	err = svc.fileStore.Delete(&files[0])
	return response, err
}

// Download gets a file from given fileManager and returns io.ReadCloser as part of Attachment Response
func (svc *service) Download(ctx context.Context, request downloadRequest) (response.Attachment, error) {
	response := response.Attachment{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	files, err := svc.fileStore.List(&file.Criteria{
		ID: request.ID,
	})
	if len(files) == 0 {
		svc.logger.Error("msg", err)
		return response, errors.New("couldn't get file")
	} else if files[0].Username != u.Username {
		if ok := u.Can(permission.ListCampaignFiles); !ok {
			return response, errs.ForbiddenError{"user doesn't have permission to list campaign files"}
		}
	}
	response.ReadCloser, err = svc.fileManager.Open(filepath.Join(files[0].Username, files[0].LocalName))
	return response, err
}

// List lets user list and filter uploaded files
func (svc *service) List(ctx context.Context, request listRequest) (listResponse, error) {
	response := listResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	if request.Username != u.Username {
		if ok := u.Can(permission.ListCampaignFiles); !ok {
			return response, errs.ForbiddenError{"user doesn't have permission to list campaign files"}
		}
	}
	response.Files, err = svc.fileStore.List(&request.Criteria)
	return response, err
}

// Upload lets us upload a campaign file
func (svc *service) Upload(ctx context.Context, request uploadRequest) (uploadResponse, error) {
	response := uploadResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	f := file.File{
		Description: request.Description,
		Username:    u.Username,
		SubmittedAt: time.Now().UTC().Unix(),
		Name:        request.FileName,
	}
	f.LocalName = f.Name + svc.randFunc()
	writer, err := svc.fileManager.Open(filepath.Join(u.Username, f.LocalName))
	id, err := svc.fileStore.Save(&f, svc.processExcelFunc, request.ReadCloser, writer)
	response.ID = id
	return response, nil
}
