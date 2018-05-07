package users

import (
	"context"

	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"github.com/pkg/errors"
)

// Service is user service's interface
type Service interface {
	Permissions(ctx context.Context, request permissionsRequest) (permissionsResponse, error)
	Edit(ctx context.Context, request editRequest) (editResponse, error)
	Add(ctx context.Context, request addRequest) (addResponse, error)
	List(ctx context.Context, request listRequest) (listResponse, error)
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

func (s *service) Permissions(ctx context.Context, request permissionsRequest) (permissionsResponse, error) {
	response := permissionsResponse{}
	response.Permissions = permission.GetList()
	return response, nil
}

func (s *service) Edit(ctx context.Context, request editRequest) (editResponse, error) {
	response := editResponse{}
	u, err := s.userStore.Get(request.Username)
	if err != nil {
		err = errors.Wrap(errs.ErrorResponse{
			Errors: []errs.ResponseError{
				{
					Type:    errs.ErrorTypeRequest,
					Message: "couldn't get user",
				},
			},
		}, err.Error())
		return response, err
	}

	if request.Name != "" {
		u.Name = request.Name
	}
	if request.Email != "" {
		u.Email = request.Email
	}
	if request.ConnectionGroup != "" {
		u.ConnectionGroup = request.ConnectionGroup
	}
	if request.Password != "" {
		u.Password = request.Password
	}
	if len(request.Permissions) > 0 {
		u.Permissions = request.Permissions
	}
	if request.Suspended == true {
		u.Suspended = true
	}
	if u.Suspended == true && request.Suspended == false {
		u.Suspended = false
	}

	err = u.Validate()
	if err != nil {
		verrs := err.(*errs.ValidationError).Errors
		errResp := errs.ErrorResponse{}
		errResp.Ok = false
		errResp.Errors = make([]errs.ResponseError, len(verrs))
		for k, v := range verrs {
			errResp.Errors = append(errResp.Errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		errResp.Request = request
		return response, errResp
	}
	err = s.userStore.Update(u, len(request.Password) > 1)
	if err != nil {
		err = errors.Wrap(errs.ErrorResponse{
			Errors: []errs.ResponseError{
				{
					Type:    errs.ErrorTypeDB,
					Message: "couldn't update user",
				},
			},
		}, err.Error())
		return response, err
	}
	response.User = u
	return response, nil

}

func (s *service) Add(ctx context.Context, request addRequest) (addResponse, error) {
	response := addResponse{}
	u := &user.User{
		Email:           request.Email,
		ConnectionGroup: request.ConnectionGroup,
		Username:        request.Username,
		Password:        request.Password,
		Name:            request.Name,
		Permissions:     request.Permissions,
		RegisteredAt:    time.Now().UTC().Unix(),
		Suspended:       request.Suspended,
	}
	err := u.Validate()
	if err != nil {
		verrs := err.(*errs.ValidationError).Errors
		errResp := errs.ErrorResponse{}
		errResp.Ok = false
		errResp.Errors = make([]errs.ResponseError, len(verrs))
		for k, v := range verrs {
			errResp.Errors = append(errResp.Errors, errs.ResponseError{
				Type:    errs.ErrorTypeForm,
				Message: v,
				Field:   k,
			})
		}
		errResp.Request = request
		return response, errResp
	}
	id, err := s.userStore.Add(u)
	if err != nil {
		err = errors.Wrap(errs.ErrorResponse{
			Errors: []errs.ResponseError{
				{
					Type:    errs.ErrorTypeDB,
					Message: "couldn't add user",
				},
			},
		}, err.Error())
		return response, err
	}
	response.ID = id
	return response, nil
}

func (s *service) List(ctx context.Context, request listRequest) (listResponse, error) {
	response := listResponse{}
	users, err := s.userStore.List(request.Criteria)
	if err != nil {
		err = errors.Wrap(errs.ErrorResponse{
			Errors: []errs.ResponseError{
				{
					Type:    errs.ErrorTypeDB,
					Message: "couldn't get users",
				},
			},
		}, err.Error())
		return response, err
	}
	response.Users = users
	return response, nil
}
