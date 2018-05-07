package campaign

import (
	"context"

	"bitbucket.org/codefreak/hsmpp/smpp/entities/campaign"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/services"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/numfile"
	"github.com/pkg/errors"
)

// Service is interface for campaign service
type Service interface {
	List(ctx context.Context, request listRequest) (listResponse, error)
	Start(ctx context.Context, request startRequest) (startResponse, error)
	Progress(ctx context.Context, request progressRequest) (progressResponse, error)
	Stop(ctx context.Context, request stopRequest) (stopResponse, error)
}

type service struct {
	logger        logger.Logger
	campaignStore campaign.Store
	authenticator user.Authenticator
}

// NewService returns a new user service
func NewService(logger logger.Logger, campaignStore campaign.Store, auth user.Authenticator) Service {
	return &service{
		logger, campaignStore, auth,
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
	if request.Mask {
		u, err := user.FromContext(ctx)
		if err != nil {
			return response, err
		}
		if !u.Can(permission.Mask) {
			return response, errs.ForbiddenError{"user doesn't have mask permissions"}
		}
	}
	var numbers []numfile.Row
	if request.FileID != 0 {
		var files []numfile.NumFile
		files, err := numfile.List(numfile.Criteria{
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
		numbers, err = files[0].ToNumbers(&numfile.RealNumFileIO{})
		if err != nil {
			log.WithError(err).Error("Couldn't read numbers from file.")
			resp := services.ClientResponse{}
			resp.Errors = []services.ResponseError{
				{
					Type:    services.ErrorTypeForm,
					Message: "Couldn't read numbers from file.",
					Field:   "FileID",
				},
			}
			resp.Send(w, *r, http.StatusInternalServerError)
		}
	} else if uReq.Numbers != "" {
		numbers = numfile.RowsFromString(uReq.Numbers)
	} else {
		log.WithError(err).Error("No numbers provided.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeRequest,
				Message: "No numbers provided. You should either select a file or send comma separated list of numbers",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
	}
	response.ID, err := svc.campaignStore.Save()
}

func (svc *service) Progress(ctx context.Context, request progressRequest) (progressResponse, error) {

}

func (svc *service) Stop(ctx context.Context, request stopRequest) (stopResponse, error) {

}
