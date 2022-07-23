package main

import (
	"log"
	"net/http"

	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	mux := http.NewServeMux()
	// The generated constructors return a path and a plain net/http
	// handler.
	mux.Handle(apiv1connect.NewIpamServiceHandler(service.New()))
	err := http.ListenAndServe(
		"localhost:8080",
		// For gRPC clients, it's convenient to support HTTP/2 without TLS. You can
		// avoid x/net/http2 by using http.ListenAndServeTLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
	log.Fatalf("listen failed: %v", err)
	return nil
}
