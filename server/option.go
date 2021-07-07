package server

import (
	"context"
	"net/http"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/cnative/pkg/log"

	"github.com/cnative/pkg/health"

	"github.com/cnative/pkg/auth"
)

type (
	// Option configures choices
	Option interface {
		apply(*runtime)
	}
	optionFunc func(*runtime)
)

// Probes used by runtime to check health
func Probes(probes map[string]health.Probe) Option {
	return optionFunc(func(r *runtime) {
		r.probes = probes
	})
}

// Logger for runtime
func Logger(l log.Logger) Option {
	return optionFunc(func(r *runtime) {
		r.logger = l.NamedLogger("rt")
	})
}

// Debug enables debug and sets up port for http/pprof data
func Debug(debug bool, port uint) Option {
	return optionFunc(func(r *runtime) {
		r.debugEnabled = debug
		r.dPort = port
	})
}

// Tags name value label pairs that is applied to server for info purpuse
func Tags(tags map[string]string) Option {
	return optionFunc(func(r *runtime) {
		r.tags = tags
	})
}

// AuthRuntime sets up AuthN and AuthZ for server runtime
func AuthRuntime(authRuntime auth.Runtime) Option {
	return optionFunc(func(r *runtime) {
		r.authRuntime = authRuntime
	})
}

// GRPCPort of the main grpc server
func GRPCPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.gPort = port
	})
}

// GRPCServerKeepAlive grpc server connection keep alive properties
func GRPCServerKeepAlive(ka *keepalive.ServerParameters) Option {
	return optionFunc(func(r *runtime) {
		r.grpcServerKAProps = ka
	})
}

// HTTPPort of the main http server
func HTTPPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.htPort = port
	})
}

// HealthPort for health check
func HealthPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.hPort = port
	})
}

// TLSCred Key and Cert Files
func TLSCred(certFile, keyFile, clientCA string) Option {
	return optionFunc(func(r *runtime) {
		r.keyFile = keyFile
		r.certFile = certFile
		r.clientCA = clientCA
	})
}

// OTLPCollectorEP OTLP Collector endpoint
func OTLPCollectorEP(ep string) Option {
	return optionFunc(func(r *runtime) {
		r.otlpCollectorEP = ep
	})
}

// OTLPCollectorEP OTLP Collector endpoint
func OTLPCollectorTLSCred(cred credentials.TransportCredentials) Option {
	return optionFunc(func(r *runtime) {
		r.otlpCollectorTLSCred = cred
	})
}

// GRPCAPIHandlers sets up grpc API handlers needs to be registered with Runtime
func GRPCAPIHandlers(handler GRPCAPIHandler, handlers ...GRPCAPIHandler) Option {
	return optionFunc(func(r *runtime) {
		r.grpcAPIHandlers = append([]GRPCAPIHandler{handler}, handlers...)
		r.grpcEnabled = true
	})
}

//GRPCGateway option to enable HTTP REST API Gateway for the gRPC apis.
func GRPCGateway() Option {
	return optionFunc(func(r *runtime) {
		r.gwEnabled = true
	})
}

// HTTPAPI that needs to be registered with Runtime
func HTTPAPI(handler http.Handler) Option {
	return optionFunc(func(r *runtime) {
		r.httpHandler = handler
		r.htEnabled = true
	})
}

// ShutdownHook called in the after shutting all the support services
func ShutdownHook(hook func(context.Context) error) Option {
	return optionFunc(func(r *runtime) {
		r.shutdownHook = hook
	})
}

// Daemon a is background service with no listener
func Daemon(daemon DaemonHandler) Option {
	return optionFunc(func(r *runtime) {
		r.daemon = daemon
	})
}
