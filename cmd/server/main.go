package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime/debug"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/pkg/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type config struct {
	GrpcServerEndpoint string
	Log                *zap.SugaredLogger
}
type server struct {
	c    config
	port int
	log  *zap.SugaredLogger
}

func newServer(c config) *server {
	return &server{
		c:   c,
		log: c.Log,
	}
}
func (s *server) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcLogger := s.log.Named("grpc").Desugar()
	grpc_zap.ReplaceGrpcLoggerV2(grpcLogger)

	recoveryOpt := grpc_recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p any) error {
			grpcLogger.Sugar().Errorf("[PANIC] %s stack:%s", p, string(debug.Stack()))
			return status.Errorf(codes.Internal, "%s", p)
		},
	)

	opts := []grpc.ServerOption{
		// FIXME enable TLS for all incoming connections.
		// grpc.Creds(creds),

		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(grpcLogger),
			grpc_recovery.StreamServerInterceptor(recoveryOpt),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(grpcLogger),
			grpc_recovery.UnaryServerInterceptor(recoveryOpt),
		)),
	}
	server := grpc.NewServer(opts...)

	reflection.Register(server)
	v1.RegisterIpamServiceServer(server, service.New())

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return nil
}
