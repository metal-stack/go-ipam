package test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
)

// NewTestServer can be used in unit- and integration tests to have a ipam service running with a memory backend
func NewTestServer(ctx context.Context, log *slog.Logger) (apiv1connect.IpamServiceClient, func()) {
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(log, goipam.New(ctx)),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	closer := func() {
		server.Close()
	}

	connectClient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)
	return connectClient, closer
}
