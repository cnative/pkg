package server

import (
	"context"
	"fmt"
	"net/http"

	"go.opencensus.io/stats/view"
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
func Logger(l *log.Logger) Option {
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

// MetricsPort of the main grpc server
func MetricsPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.mPort = port
	})
}

// Trace enable/disable
func Trace(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.traceEnabled = enabled
	})
}

// OCAgentEP Opencensus Agent End point
func OCAgentEP(host string, port uint) Option {
	return optionFunc(func(r *runtime) {
		r.ocAgentEP = fmt.Sprintf("%s:%d", host, port)
	})
}

// OCAgentNamespace used for isolation/categorization
func OCAgentNamespace(ns string) Option {
	return optionFunc(func(r *runtime) {
		r.ocAgentNamespace = ns
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

// GatewayClientTLSCred Key and Cert file to be used by gateway client to connect to the gw server
func GatewayClientTLSCred(certFile, keyFile string) Option {
	return optionFunc(func(r *runtime) {
		r.gwClientKeyFile = keyFile
		r.gwClientCertFile = certFile
	})
}

// GRPCAPI that needs to be registered with Runtime
func GRPCAPI(handler GRPCAPIHandler, gw bool) Option {
	return optionFunc(func(r *runtime) {
		r.grpcAPIHandler = handler
		r.grpcEnabled = true
		r.gwEnabled = gw
	})
}

// HTTPAPI that needs to be registered with Runtime
func HTTPAPI(handler http.Handler) Option {
	return optionFunc(func(r *runtime) {
		r.httpHandler = handler
		r.htEnabled = true
	})
}

// CustomMetricsViews custom metrics
func CustomMetricsViews(views ...*view.View) Option {
	return optionFunc(func(r *runtime) {
		r.statsViews = views
	})
}

// ProcessMetrics ebable collection of process metrics
func ProcessMetrics(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.processMetricsEnabled = enabled
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
