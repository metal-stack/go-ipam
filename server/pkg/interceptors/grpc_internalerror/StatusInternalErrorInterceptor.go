package grpc_internalerror

import (
	"context"
	"github.com/gogo/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

/*
	Every response with an "plain" error will be converted to an status-error with codes.Internal.
*/

// UnaryServerInterceptor returns a new unary server interceptor for panic recovery.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {

		result, err := handler(ctx, req)
		_, ok := status.FromError(err)
		if !ok {
			err = status.Error(codes.Internal, err.Error())
		}

		return result, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func StreamServerInterceptor() grpc.StreamServerInterceptor {

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

		err = handler(srv, stream)
		_, ok := status.FromError(err)
		if !ok {
			err = status.Error(codes.Internal, err.Error())
		}

		return err
	}
}
