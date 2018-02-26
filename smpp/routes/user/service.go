package user

import (
	"context"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
)

// Service is user service's interface
type Service interface {
	Info(ctx context.Context, request infoRequest) (infoResponse, error)
	Edit(ctx context.Context, request editRequest) (editResponse, error)
}

type service struct {
	db         *db.DB
	logger     logger.Logger
	tokenStore tokenStorer
	userStore  userStorer
	hashFunc   func(string) (string, error)
}

// NewService returns a new user service
func NewService(db *db.DB, logger logger.Logger, tokenStore tokenStorer, userStore userStorer, hashFunc func(string) (string, error)) Service {
	return &service{
		db, logger, tokenStore, userStore,hashFunc
	}
}

// Info endpoint returns info of user in current context
func (s *service) Info(ctx context.Context, request infoRequest) (infoResponse, error) {
	response := infoResponse{}
	u, err := fromContext(ctx)
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
	u, err := fromContext(ctx)
	if err != nil {
		return response, err
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
		vErr := err.(*validationError)
		errResp := routes.ErrorResponse{}
		for k, v := range vErr.Errors {
			errResp.Errors = append(errResp.Errors, routes.ResponseError{
				Type:    routes.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		return response, errResp
	}
	err = s.userStore.Update(u, s.hashFunc, len(request.Password) > 1)
	if err != nil {
		s.logger.Error("msg", "couldn't update user", "error", err)
		return response, err
	}
	response.User = u
	return response, nil
}
