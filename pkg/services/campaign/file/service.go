package file

import (
	"context"

	"path/filepath"

	"bitbucket.org/codefreak/hsmpp/pkg/entities/campaign/file"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/user"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/errs"
	"bitbucket.org/codefreak/hsmpp/pkg/logger"
	"bitbucket.org/codefreak/hsmpp/pkg/response"
	"github.com/pkg/errors"
)

// Service is interface for campaign service
type Service interface {
	Delete(ctx context.Context, request deleteRequest) (deleteResponse, error)
	Download(ctx context.Context, request downloadRequest) (response.Attachment, error)
}

type service struct {
	logger        logger.Logger
	fileStore     file.Store
	fileManager   file.OpenReadWriteCloser
	authenticator user.Authenticator
}

// NewService returns a new user service
func NewService(logger logger.Logger, fileStore file.Store, fileManager file.OpenReadWriteCloser, auth user.Authenticator) Service {
	return &service{
		logger,
		fileStore, fileManager,
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
