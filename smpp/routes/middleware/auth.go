package middleware

import (
	"context"

	"bytes"
	"encoding/base64"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/entities/user"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
)

// AuthMiddleware returns a Basic Authentication middleware for a particular user and password.
func AuthMiddleware(authority user.Authenticator, realm string, actions ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			auth, ok := ctx.Value(httptransport.ContextKeyRequestAuthorization).(string)
			if !ok {
				return nil, errs.AuthError{}
			}

			givenUser, givenPassword, ok := parseBasicAuth(auth)
			if !ok {
				return nil, errs.AuthError{}
			}

			ctx, authzr, err := authority.Authenticate(ctx, stringutils.ByteToString(givenUser), stringutils.ByteToString(givenPassword))
			if err != nil {
				return nil, errors.Wrap(errs.AuthError{}, err.Error())
			}
			ok = authzr.Can(actions...)
			if !ok {
				return nil, errors.Wrap(errs.ForbiddenError{}, "permission denied")
			}

			return next(ctx, request)
		}
	}
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ([]byte("Aladdin"), []byte("open sesame"), true).
func parseBasicAuth(auth string) (username, password []byte, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}

	s := bytes.IndexByte(c, ':')
	if s < 0 {
		return
	}
	return c[:s], c[s+1:], true
}
