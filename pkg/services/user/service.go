package user

import (
	"context"

	"bitbucket.org/codefreak/hsmpp/pkg/entities/user"
	"bitbucket.org/codefreak/hsmpp/pkg/errs"
	"bitbucket.org/codefreak/hsmpp/pkg/logger"
)

// Service is user service's interface
type Service interface {
	Info(ctx context.Context, request infoRequest) (infoResponse, error)
	Edit(ctx context.Context, request editRequest) (editResponse, error)
}

type service struct {
	logger        logger.Logger
	userStore     user.Store
	authenticator user.Authenticator
}

// NewService returns a new user service
func NewService(logger logger.Logger, userStore user.Store, authenticator user.Authenticator) Service {
	return &service{
		logger, userStore, authenticator,
	}
}

// Info endpoint returns info of user in current context
func (s *service) Info(ctx context.Context, request infoRequest) (infoResponse, error) {
	response := infoResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, err
	}
	response.ConnectionGroup = u.ConnectionGroup
	response.Permissions = u.Permissions
	response.Suspended = u.Suspended
	response.RegisteredAt = u.RegisteredAt
	response.Username = u.Username
	response.Name = u.Name
	return response, nil
}

// Edit endpoint edits user in current context
func (s *service) Edit(ctx context.Context, request editRequest) (editResponse, error) {
	response := editResponse{}
	u, err := user.FromContext(ctx)
	if err != nil {
		return response, errs.BadRequestError(err)
	}
	if request.Name != "" {
		u.Name = request.Name
	}
	if request.Email != "" {
		u.Email = request.Email
	}
	if request.Password != "" {
		u.Password = request.Password
	}
	err = u.Validate()
	if err != nil {
		vErr := err.(*errs.ValidationError)
		errResp := errs.ErrorResponse{}
		for k, v := range vErr.Errors {
			errResp.Errors = append(errResp.Errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		return response, errResp
	}
	err = s.userStore.Update(u, len(request.Password) > 1)
	if err != nil {
		s.logger.Error("msg", "couldn't update user", "error", err)
		return response, err
	}
	response.User = u
	return response, nil
}
