package middleware

import (
	"context"
	"time"

	"github.com/haisum/smpp-app/pkg/logger"
	"github.com/go-kit/kit/endpoint"
)

// LoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func LoggingMiddleware(logger logger.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				logger.Info("transport_error", err, "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)

		}
	}
}
