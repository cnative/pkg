package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cnative/pkg/auth"
	"github.com/cnative/pkg/health"
	"github.com/cnative/pkg/log"
	"github.com/cnative/pkg/server/middleware"
	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// default process metrics collection frequency
const defaultProcessMetricsCollectionFrequency = 5 * time.Second

type (

	// GRPCAPIHandler handles api registration with the grpc server
	GRPCAPIHandler interface {
		Register(context.Context, *grpc.Server, *grpc_runtime.ServeMux, *grpc.ClientConn) error
		io.Closer
	}

	runtime struct {
		logger       log.Logger
		probes       map[string]health.Probe
		grpcServer   *grpc.Server
		gwServer     *http.Server
		healthServer health.Service
		debugServer  *http.Server
		htServer     *http.Server
		httpHandler  http.Handler
		daemon       DaemonHandler

		gwClientConn *grpc.ClientConn

		grpcServerKAProps     *keepalive.ServerParameters
		authRuntime           auth.Runtime
		grpcAPIHandlers       []GRPCAPIHandler
		grpcMethodDescriptors map[string]*desc.MethodDescriptor

		gPort  uint // GRPC server port
		htPort uint // HTTP server port
		hPort  uint // health server port
		dPort  uint // debug server port

		certFile string // TLS certificate used by server listener
		keyFile  string // TLS private key used by server listener
		clientCA string // mTLS. if specified connections are accepted from clients that present certs signed by this CA

		grpcEnabled  bool // enable grpc server
		htEnabled    bool // enable http server
		gwEnabled    bool // enable gateway server
		debugEnabled bool // if enabled serve pprof data via HTTP server

		otlpCollectorEP      string                           // OTLP collector endpoint to which the metrics and trace data is exported
		otlpCollectorTLSCred credentials.TransportCredentials // OTLP collector TLS certificate used by client
		otlpController       *basic.Controller                // OTLP controller

		tags         map[string]string // info purpose labels
		startTime    time.Time
		shutdownHook func(context.Context) error // shutdown hook for runtime
	}

	//Runtime interface defines server operations
	Runtime interface {
		Start(context.Context) (chan error, error)
		Stop(context.Context)
	}

	// DaemonHandler for running tasks in the background that does not have http or grpc interfaces
	DaemonHandler interface {
		Serve(context.Context) error
		Stop(context.Context) error
	}
)

func (f optionFunc) apply(r *runtime) {
	f(r)
}

func (r *runtime) isSecureConnection() bool {
	return r.keyFile != "" && r.certFile != ""
}

func (r *runtime) wrapListenerWithTLS(l net.Listener) (net.Listener, error) {
	tc, err := r.getTLSConfig()
	if err != nil {
		return nil, err
	}

	return tls.NewListener(l, tc), nil
}

// NewRuntime returns a new Runtime
func NewRuntime(ctx context.Context, name string, options ...Option) (Runtime, error) {
	// setup defaults
	r := &runtime{}
	for _, opt := range options {
		opt.apply(r)
	}
	if r.logger == nil {
		r.logger = log.NewNop()
	}

	r.logger.Infow("TLS info", "key-file", r.keyFile, "cert-file", r.certFile, "client-ca", r.clientCA)
	if !r.isSecureConnection() {
		r.logger.Warn("no TLS key specified. starting server insecurely....")
	}

	r.healthServer = health.New(health.BindPort(r.hPort), health.Logger(r.logger))

	if r.debugEnabled {
		r.debugServer = &http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", r.dPort),
			Handler: getDebugHandler(r),
		}
	}

	if r.grpcEnabled {
		r.logger.Debug("creating grpc server")
		r.grpcMethodDescriptors = map[string]*desc.MethodDescriptor{}
		gsrv, err := r.newGRPCServer()
		if err != nil {
			return nil, err
		}

		r.grpcServer = gsrv
		var gwmux *grpc_runtime.ServeMux
		if r.gwEnabled {
			r.logger.Info("grpc gateway enabled")
			gwmux = grpc_runtime.NewServeMux(grpc_runtime.WithMarshalerOption(grpc_runtime.MIMEWildcard, &grpc_runtime.JSONPb{}))
			r.gwServer = &http.Server{Handler: otelhttp.NewHandler(gwmux, "ggw")}
			conn, err := r.getGRPCClientConnectionForGateway(ctx)
			if err != nil {
				return nil, err
			}
			r.gwClientConn = conn
		} else {
			r.logger.Info("grpc gateway not enabled")
		}

		if len(r.grpcAPIHandlers) == 0 {
			return nil, errors.Errorf("no grpc handlers registered. expect atleast one")
		}

		for _, h := range r.grpcAPIHandlers {
			if err := h.Register(ctx, r.grpcServer, gwmux, r.gwClientConn); err != nil {
				return nil, err
			}
		}

		sds, _ := grpcreflect.LoadServiceDescriptors(r.grpcServer)
		for _, sd := range sds {
			for _, md := range sd.GetMethods() {
				methodName := fmt.Sprintf("/%s/%s", sd.GetFullyQualifiedName(), md.GetName())
				r.grpcMethodDescriptors[methodName] = md
			}
		}
	}

	if r.htEnabled {
		r.logger.Info("http server enabled")
		r.htServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", r.htPort),
			Handler: otelhttp.NewHandler(r.httpHandler, "ht"),
		}
	}

	return r, nil
}

func (r *runtime) startOTLPExporter(ctx context.Context) error {

	tlsOpt := otlpgrpc.WithInsecure()
	if r.otlpCollectorTLSCred != nil {
		tlsOpt = otlpgrpc.WithTLSCredentials(r.otlpCollectorTLSCred)
	}
	driver := otlpgrpc.NewDriver(tlsOpt, otlpgrpc.WithEndpoint(r.otlpCollectorEP))

	_, _, ctrl, err := otlp.InstallNewPipeline(ctx, driver)
	if err != nil {
		return err
	}
	global.SetMeterProvider(ctrl.MeterProvider())
	r.otlpController = ctrl

	return nil
}

// Start server runtime
func (r *runtime) Start(ctx context.Context) (chan error, error) {

	errc := make(chan error, 8) // error buffer channel for goroutines below

	// Shutdown on SIGINT, SIGTERM
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// Start http listener that exposes server pprof runtime data
	if r.debugEnabled {
		go func() {
			r.logger.Infow("starting debug server", "port", r.dPort)
			err := r.debugServer.ListenAndServe()
			errc <- errors.Wrap(err, "debug server returned an error")
		}()
	}

	if r.otlpCollectorEP != "" {
		// start otlp exporter if  specified
		if err := r.startOTLPExporter(ctx); err != nil {
			return nil, err
		}
	}

	var cm, tcm cmux.CMux
	if r.grpcEnabled {
		// start gRPC server
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.gPort))
		if err != nil {
			r.logger.Errorf("failed to create grpc listener -%v ", err)
			return nil, err
		}
		cm = cmux.New(lis)
		var grpcL, gwL net.Listener
		if r.isSecureConnection() {
			tlsl := cm.Match(cmux.TLS())
			tlsl, err = r.wrapListenerWithTLS(tlsl)
			if err != nil {
				return nil, err
			}
			tcm = cmux.New(tlsl)
			grpcL = tcm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
			gwL = tcm.Match(cmux.HTTP1Fast("PATCH")) // include PATCH as well. https://github.com/soheilhy/cmux/blob/master/matchers.go#L46
		} else {
			gwL = cm.Match(cmux.HTTP1Fast("PATCH"))
			grpcL = cm.Match(cmux.Any())
		}

		go func() {
			r.logger.Infow("starting grpc server", "port", r.gPort)
			err := r.grpcServer.Serve(grpcL)
			errc <- errors.Wrap(err, "grpc server returned an error")
		}()
		if r.gwEnabled {
			// start gRPC gateway
			go func() {
				r.logger.Infow("starting gateway server", "port", r.gPort)
				err := r.gwServer.Serve(gwL)
				errc <- errors.Wrap(err, "grpc gateway server returned an error")
			}()
		}
	}

	if r.htEnabled {
		// start HTTP server
		go func() {
			r.logger.Infow("starting http server", "port", r.htPort)
			var err error
			if r.isSecureConnection() {
				err = r.htServer.ListenAndServeTLS(r.certFile, r.keyFile)
			} else {
				err = r.htServer.ListenAndServe()
			}
			errc <- errors.Wrap(err, "http server returned an error")
		}()
	}

	// Start health server
	go func() {
		r.logger.Infow("starting health service", "port", r.hPort)
		for name, probe := range r.probes {
			r.healthServer.RegisterProbe(name, probe)
		}
		err := r.healthServer.Start()
		errc <- errors.Wrap(err, "health service returned an error")
	}()

	if r.daemon != nil {
		// Start daemon server
		go func() {
			r.logger.Info("starting daemnon server")
			errc <- r.daemon.Serve(ctx)
		}()
	}

	if cm != nil {
		if tcm != nil {
			go func() {
				errc <- tcm.Serve() // cmux tls
			}()
		}
		go func() {
			errc <- cm.Serve() // cmux
		}()
	}

	r.startTime = time.Now()
	return errc, nil
}

// Stop server runtime
func (r *runtime) Stop(ctx context.Context) {

	r.logger.Infof("shutting down..")
	for _, h := range r.grpcAPIHandlers {
		h.Close()
	}

	if r.gwEnabled {
		r.logger.Info("shutting gateway server")
		if err := r.gwClientConn.Close(); err != nil {
			r.logger.Errorf("error happened while closing gateway grpc client -%v", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.gwServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting gateway server -%v", err)
		}
	}

	if r.grpcEnabled {
		// gracefully shutdown the gRPC server
		r.logger.Info("shutting grpc server")
		r.grpcServer.GracefulStop()
	}

	if r.htEnabled {
		r.logger.Info("shutting HTTP server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.htServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting HTTP server -%v", err)
		}
	}

	// gracefully shutdown the health server
	r.logger.Info("shutting health server")
	if err := r.healthServer.Stop(ctx); err != nil {
		r.logger.Fatalf("error shutting down health server %v ", err)
	}

	if r.debugEnabled {
		r.logger.Info("shutting debug server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.debugServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting debug server -%v", err)
		}
	}

	if r.daemon != nil {
		r.logger.Info("stopping daemon server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.daemon.Stop(ctx); err != nil {
			r.logger.Errorf("error happened while stopping daemon server", err)
		}
	}

	if r.shutdownHook != nil {
		r.logger.Info("calling shutdown hook")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.shutdownHook(ctx); err != nil {
			r.logger.Errorf("error happened while calling shutdown hook", err)
		}
	}

	if r.otlpController != nil {
		r.logger.Info("shutting otlp controller")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.otlpController.Stop(ctx); err != nil {
			r.logger.Errorf("error happened while stopping otlp controller", err)
		}
	}
}

// grpc server connection keep alive properties
func defaultServerKeepAliveConnectionProps() keepalive.ServerParameters {
	return keepalive.ServerParameters{
		MaxConnectionIdle:     60 * time.Second, // If a client is idle for 60 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 15 * time.Second, // Allow 15 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  60 * time.Second, // Ping the client if it is idle for 60 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
		MaxConnectionAge:      4 * time.Hour,    // If any connection is alive for more than 4 Hours, send a GOAWAY
	}
}

func (r *runtime) newGRPCServer() (*grpc.Server, error) {
	r.logger.Debug("creating new gRPC server")

	var sacProp keepalive.ServerParameters
	if r.grpcServerKAProps != nil {
		sacProp = *r.grpcServerKAProps
	} else {
		sacProp = defaultServerKeepAliveConnectionProps()
	}

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(sacProp),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
			PermitWithoutStream: true,            // Allow pings even when there are no active streams
		}),
	}

	uis := []grpc.UnaryServerInterceptor{otelgrpc.UnaryServerInterceptor()}
	sis := []grpc.StreamServerInterceptor{otelgrpc.StreamServerInterceptor()}
	if r.authRuntime != nil {
		ui, si := middleware.GRPCAuthInterceptors(r.authRuntime, r.grpcMethodDescriptors)
		uis, sis = append(uis, ui), append(sis, si)
	} else {
		r.logger.Warn("auth runtime not enabled for the server")
	}
	opts = append(opts, middleware.GRPCUnaryInterceptors(uis...)...)
	opts = append(opts, middleware.GRPCStreamInterceptors(sis...)...)
	return grpc.NewServer(opts...), nil
}

// get TLS Config
func (r *runtime) getTLSConfig() (*tls.Config, error) {
	// Load the certificates from disk
	certificate, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{certificate},
	}

	if r.clientCA != "" {
		// Create a certificate pool from the certificate authority
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(r.clientCA)
		if err != nil {
			return nil, err
		}

		// Append the client certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return nil, err
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = certPool
	} else {
		r.logger.Info("mTLS not enabled")
	}

	return &tlsConfig, nil
}

func (r *runtime) getGRPCClientConnectionForGateway(ctx context.Context) (*grpc.ClientConn, error) {
	grpc.SendHeader(ctx, metadata.Pairs("content-type", "application/grpc"))
	opts := []grpc.DialOption{}

	if r.isSecureConnection() {
		tc, err := newTLSConfig()
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tc)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	addr := fmt.Sprintf("127.0.0.1:%d", r.gPort)
	return grpc.Dial(addr, opts...)
}
