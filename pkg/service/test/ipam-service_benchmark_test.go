package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bufbuild/connect-go"
	compress "github.com/klauspost/connect-compress"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"go.uber.org/zap/zaptest"
)

// BenchmarkGrpcImpact located in a separate package to prevent import cycles.
func BenchmarkGrpcImpact(b *testing.B) {

	ipam := goipam.New()

	// Get client and server options for all compressors...
	clientOpts, serverOpts := compress.All(compress.LevelBalanced)

	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(zaptest.NewLogger(b).Sugar(), ipam),
		serverOpts,
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	grpc := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		connect.WithGRPC(),
		clientOpts,
	)
	httpclient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		clientOpts,
	)

	benchmarks := []struct {
		name string
		f    func()
	}{
		{
			name: "library",
			f: func() {
				p, err := ipam.NewPrefix("192.168.0.0/24")
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = ipam.DeletePrefix(p.Cidr)
				if err != nil {
					panic(err)
				}
			},
		},
		{
			name: "grpc",
			f: func() {
				p, err := grpc.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = grpc.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
			},
		},
		{
			name: "http",
			f: func() {
				p, err := httpclient.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = httpclient.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.f()
			}
		})
	}
	for n := 0; n < b.N; n++ {

	}
}
