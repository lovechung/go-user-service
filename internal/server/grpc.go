package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/lovechung/api-base/api/user"
	contrib "github.com/lovechung/go-kit/contrib/metrics"
	"github.com/lovechung/go-kit/middleware/metrics"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"user-service/internal/conf"
	"user-service/internal/service"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, us *service.UserService, logger log.Logger) *grpc.Server {
	meter := global.Meter("user-service")
	requestHistogram, _ := meter.SyncInt64().Histogram("user_service_req", instrument.WithUnit(unit.Milliseconds))

	var opts = []grpc.ServerOption{
		grpc.Middleware(
			middleware.Chain(
				recovery.Recovery(),
				ratelimit.Server(),
				tracing.Server(),
				metrics.Server(
					metrics.WithSeconds(contrib.NewHistogram(requestHistogram)),
				),
				//jwt.Server(func(token *jwtV4.Token) (interface{}, error) {
				//	return []byte("123456"), nil
				//}),
				logging.Server(logger),
			),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	user.RegisterUserServer(srv, us)
	return srv
}
