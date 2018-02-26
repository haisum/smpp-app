package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"

	"bytes"
	"encoding/base64"
	"strings"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
)

// AuthError represents an authorization error.
type AuthError struct {
	Realm string
}

// StatusCode is an implementation of the StatusCoder interface in go-kit/http.
func (AuthError) StatusCode() int {
	return http.StatusUnauthorized
}

// Error is an implementation of the Error interface.
func (AuthError) Error() string {
	return http.StatusText(http.StatusUnauthorized)
}

// Headers is an implementation of the Headerer interface in go-kit/http.
func (e AuthError) Headers() http.Header {
	return http.Header{
		"Content-Type":           []string{"text/plain; charset=utf-8"},
		"X-Content-Type-Options": []string{"nosniff"},
		"WWW-Authenticate":       []string{fmt.Sprintf(`Basic realm=%q`, e.Realm)},
	}
}

// Returns a hash of a given slice.
func toHashSlice(s []byte) []byte {
	hash := sha256.Sum256(s)
	return hash[:]
}

type authorizer interface {
	Can(actions ...string) bool
}

type authenticator interface {
	Authenticate(HashMatchFunc func(hash, str string) bool, username, password []byte) (authorizer, error)
}

// AuthMiddleware returns a Basic Authentication middleware for a particular user and password.
func AuthMiddleware(authority authenticator, HashMatchFunc func(hash, str string) bool, realm string, actions ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			auth, ok := ctx.Value(httptransport.ContextKeyRequestAuthorization).(string)
			if !ok {
				return nil, AuthError{realm}
			}

			givenUser, givenPassword, ok := parseBasicAuth(auth)
			if !ok {
				return nil, AuthError{realm}
			}

			authorizer, err := authority.Authenticate(HashMatchFunc, givenUser, givenPassword)
			if err != nil {
				return nil, errors.Wrap(AuthError{realm}, err.Error())
			}
			ok = authorizer.Can(actions...)
			if !ok {
				return nil, errors.Wrap(AuthError{realm}, err.Error())
			}

			/*

				givenUserBytes := toHashSlice(givenUser)
				givenPasswordBytes := toHashSlice(givenPassword)
			*/
			/*

				if subtle.ConstantTimeCompare(givenUserBytes, requiredUserBytes) == 0 ||
					subtle.ConstantTimeCompare(givenPasswordBytes, requiredPasswordBytes) == 0 {
					return nil, AuthError{realm}
				}
			*/

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
