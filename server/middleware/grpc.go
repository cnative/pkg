package middleware

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/cnative/pkg/api"
	"github.com/cnative/pkg/auth"
)

type wrappedServerStream struct {
	grpc.ServerStream
	wrappedContext context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.wrappedContext
}

func wrapServerStream(stream grpc.ServerStream) *wrappedServerStream {
	if existing, ok := stream.(*wrappedServerStream); ok {
		return existing
	}
	return &wrappedServerStream{ServerStream: stream, wrappedContext: stream.Context()}
}

// Used if no interceptors are specified while chaining
func defaultUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func chainingUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	n := len(interceptors)
	switch n {
	case 0:
		return defaultUnaryInterceptor
	case 1:
		return interceptors[0]
	default: // n > 1
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

			cur := 0
			var next grpc.UnaryHandler
			next = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				if cur == n-1 {
					return handler(currentCtx, currentReq)
				}
				cur++
				resp, err := interceptors[cur](currentCtx, currentReq, info, next)
				cur--
				return resp, err
			}

			return interceptors[0](ctx, req, info, next)
		}
	}
}

// Used if no interceptors are specified while chaining
func defaultStreamInterceptor(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, stream)
}

func chainingStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	n := len(interceptors)
	switch n {
	case 0:
		return defaultStreamInterceptor
	case 1:
		return interceptors[0]
	default: // n > 1
		return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

			cur := 0
			var next grpc.StreamHandler
			next = func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				if cur == n-1 {
					return handler(currentSrv, currentStream)
				}
				cur++
				err := interceptors[cur](currentSrv, currentStream, info, next)
				cur--
				return err
			}

			return interceptors[0](srv, stream, info, next)
		}
	}
}

// WithUnaryInterceptors is a wrapper middleware that chains a set of interceptors in the specified order
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.UnaryInterceptor(chainingUnaryInterceptor(interceptors...))
}

// WithStreamInterceptors is a wrapper middleware that chains a set of interceptors in the specified order
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.ServerOption {
	return grpc.StreamInterceptor(chainingStreamInterceptor(interceptors...))
}

func auth0(ctx context.Context, authRuntime auth.Runtime, req interface{}, resource, action string) (context.Context, error) {

	token, err := getTokenFromGRPCContext(ctx)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "%v", err.Error())
	}

	ctx, c, err := authRuntime.Verify(ctx, token)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "%v", err.Error())
	}

	ctx, authzResult, err := authRuntime.Authorize(ctx, c, resource, action, req)
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "contact system administrator - %v", err.Error())
	}

	if authzResult.Allowed {
		return ctx, nil
	}

	return ctx, status.Error(codes.PermissionDenied, "contact system administrator")
}

func resourceActionResolver(methodName string, methodDescriptors map[string]*desc.MethodDescriptor) (resource string, action string, err error) {

	if dsc, ok := methodDescriptors[methodName]; ok && proto.HasExtension(dsc.GetMethodOptions(), api.E_Authz) {
		ext, err := proto.GetExtension(dsc.GetMethodOptions(), api.E_Authz)
		if err != nil {
			return "", "", err
		}
		az, ok := ext.(*api.Authz)
		if !ok {
			err = errors.Errorf("failed to type casting. expect '*api.Authz' got %T\n", az)
			return "", "", err
		}
		if az != nil {
			resource = az.Resource
			action = az.Action
		}
	}

	return resource, action, nil
}

// returns a new unary server interceptors that performs per-request auth
func unaryAuth(authRuntime auth.Runtime, methodDescriptors map[string]*desc.MethodDescriptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		resource, action, err := resourceActionResolver(info.FullMethod, methodDescriptors)
		if err != nil {
			return nil, err
		}

		newCtx, err := auth0(ctx, authRuntime, req, resource, action)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

// returns a new stream server interceptors that performs per-request auth
func streamAuth(authRuntime auth.Runtime, methodDescriptors map[string]*desc.MethodDescriptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		resource, action, err := resourceActionResolver(info.FullMethod, methodDescriptors)
		if err != nil {
			return err
		}
		newCtx, err := auth0(stream.Context(), authRuntime, stream, resource, action)
		if err != nil {
			return err
		}
		ws := wrapServerStream(stream)
		ws.wrappedContext = newCtx
		return handler(srv, ws)
	}
}

// GRPCAuth returns unary and stream interceptors
func GRPCAuth(authRuntime auth.Runtime, methodDescriptors map[string]*desc.MethodDescriptor) []grpc.ServerOption {

	return []grpc.ServerOption{
		WithUnaryInterceptors(unaryAuth(authRuntime, methodDescriptors)),
		WithStreamInterceptors(streamAuth(authRuntime, methodDescriptors)),
	}
}

// getTokenFromGRPCContext grpc token resolver
func getTokenFromGRPCContext(ctx context.Context) (string, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "Context does not contain any metadata")
	}

	authHdrs := md.Get("authorization")
	if len(authHdrs) != 1 {
		return "", errors.Errorf("Found %d authorization headers, expected 1", len(authHdrs))
	}

	sp := strings.SplitN(authHdrs[0], " ", 2)
	if len(sp) != 2 {
		return "", errors.New("authorization header has is not '<type> <token> format")
	}
	if !strings.EqualFold(sp[0], "bearer") {
		return "", errors.Errorf("Only bearer tokens are supported, not %s", sp[0])
	}

	return sp[1], nil
}
