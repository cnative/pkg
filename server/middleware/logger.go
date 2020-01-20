package middleware

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func logger(ctx context.Context) (context.Context, error) {
	//TODO
	// s := trace.FromContext(ctx)
	// if s != nil {
	// }

	return ctx, nil
}

// Logger returns a new unary server interceptor that adds logger to the context
func Logger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx, err := logger(ctx)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}
