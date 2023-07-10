package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	otelconnect "github.com/bufbuild/connect-opentelemetry-go"
	compress "github.com/klauspost/connect-compress"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"github.com/metal-stack/v"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	grpcreflect "github.com/bufbuild/connect-grpcreflect-go"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type config struct {
	GrpcServerEndpoint string
	MetricsEndpoint    string
	Log                *zap.SugaredLogger
	Storage            goipam.Storage
}
type server struct {
	c       config
	ipamer  goipam.Ipamer
	storage goipam.Storage
	log     *zap.SugaredLogger
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
	s.log.Infow("starting go-ipam", "version", v.V, "backend", s.storage.Name())

	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	// Start the prometheus HTTP server and pass the exporter Collector to it
	go s.serveMetrics()

	mux := http.NewServeMux()
	// The generated constructors return a path and a plain net/http
	// handler.
	mux.Handle(
		apiv1connect.NewIpamServiceHandler(
			service.New(s.log, s.ipamer),
			connect.WithInterceptors(
				otelconnect.NewInterceptor(otelconnect.WithMeterProvider(provider)),
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

	server := http.Server{
		Addr: s.c.GrpcServerEndpoint,
		// For gRPC clients, it's convenient to support HTTP/2 without TLS. You can
		// avoid x/net/http2 by using http.ListenAndServeTLS.
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 1 * time.Minute,
	}

	err = server.ListenAndServe()
	return err
}

func (s *server) serveMetrics() {
	s.log.Infof("serving metrics at %s/metrics", s.c.MetricsEndpoint)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(s.c.MetricsEndpoint, nil)
	if err != nil {
		fmt.Printf("error serving http: %v", err)
		return
	}
}
