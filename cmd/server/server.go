package main

import (
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // nolint:gosec
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	compress "github.com/klauspost/connect-compress/v2"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"github.com/metal-stack/v"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type config struct {
	GrpcServerEndpoint string
	MetricsEndpoint    string
	Log                *slog.Logger
	Storage            goipam.Storage
}
type server struct {
	c       config
	ipamer  goipam.Ipamer
	storage goipam.Storage
	log     *slog.Logger
}

func newServer(c config) *server {
	return &server{
		c:       c,
		ipamer:  goipam.NewWithStorage(c.Storage),
		storage: c.Storage,
		log:     c.Log,
	}
}
func (s *server) Run() error {
	s.log.Info("starting go-ipam", "version", v.V.String(), "backend", s.storage.Name())

	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	// Start the prometheus HTTP server and pass the exporter Collector to it
	go func() {
		s.log.Info("serving metrics", "at", fmt.Sprintf("%s/metrics", s.c.MetricsEndpoint))
		metricsServer := http.NewServeMux()
		metricsServer.Handle("/metrics", promhttp.Handler())
		ms := &http.Server{
			Addr:              s.c.MetricsEndpoint,
			Handler:           metricsServer,
			ReadHeaderTimeout: time.Minute,
		}
		err := ms.ListenAndServe()
		if err != nil {
			s.log.Error("unable to start metric endpoint", "error", err)
			return
		}
	}()
	go func() {
		s.log.Info("starting pprof endpoint of :2113")
		// inspect via
		// go tool pprof -http :8080 localhost:2113/debug/pprof/heap
		// go tool pprof -http :8080 localhost:2113/debug/pprof/goroutine
		server := http.Server{
			Addr:              ":2113",
			ReadHeaderTimeout: 1 * time.Minute,
		}
		err := server.ListenAndServe()
		if err != nil {
			s.log.Error("failed to start pprof endpoint", "error", err)
			return
		}
	}()

	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithMeterProvider(provider))
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	// The generated constructors return a path and a plain net/http
	// handler.
	mux.Handle(
		apiv1connect.NewIpamServiceHandler(
			service.New(s.log, s.ipamer),
			connect.WithInterceptors(
				otelInterceptor,
			),
		),
	)

	mux.Handle(grpchealth.NewHandler(
		grpchealth.NewStaticChecker(apiv1connect.IpamServiceName),
		compress.WithAll(compress.LevelBalanced),
	))
	mux.Handle(grpcreflect.NewHandlerV1(
		grpcreflect.NewStaticReflector(apiv1connect.IpamServiceName),
		compress.WithAll(compress.LevelBalanced),
	))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(
		grpcreflect.NewStaticReflector(apiv1connect.IpamServiceName),
		compress.WithAll(compress.LevelBalanced),
	))

	server := http.Server{
		Addr: s.c.GrpcServerEndpoint,
		// For gRPC clients, it's convenient to support HTTP/2 without TLS. You can
		// avoid x/net/http2 by using http.ListenAndServeTLS.
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 1 * time.Minute,
	}

	s.log.Info("started grpc server", "at", server.Addr)
	err = server.ListenAndServe()
	return err
}
