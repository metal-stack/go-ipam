package main

import (
	"net/http"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type config struct {
	GrpcServerEndpoint string
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
	s.log.Infow("starting go-ipam", "backend", s.storage.Name())
	mux := http.NewServeMux()
	// The generated constructors return a path and a plain net/http
	// handler.
	mux.Handle(apiv1connect.NewIpamServiceHandler(service.New(s.log, s.ipamer)))
	err := http.ListenAndServe(
		s.c.GrpcServerEndpoint,
		// For gRPC clients, it's convenient to support HTTP/2 without TLS. You can
		// avoid x/net/http2 by using http.ListenAndServeTLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
	return err
}
