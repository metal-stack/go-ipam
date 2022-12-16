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
	ctx := context.Background()
	ipam := goipam.New()

	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(zaptest.NewLogger(b).Sugar(), ipam),
		compress.WithAll(compress.LevelBalanced),
	))

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	grpc := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		connect.WithGRPC(),
		compress.WithAll(compress.LevelBalanced),
	)
	httpclient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		compress.WithAll(compress.LevelBalanced),
	)

	grpcUncompressed := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		connect.WithGRPC(),
	)
	httpclientUncompressed := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)

	benchmarks := []struct {
		name string
		f    func()
	}{
		{
			name: "library",
			f: func() {
				p, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = ipam.DeletePrefix(ctx, p.Cidr)
				if err != nil {
					panic(err)
				}
			},
		},
		{
			name: "grpc",
			f: func() {
				p, err := grpc.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = grpc.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
			},
		},
		{
			name: "grpc-no-compression",
			f: func() {
				p, err := grpcUncompressed.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = grpcUncompressed.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
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
				p, err := httpclient.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = httpclient.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
			},
		},
		{
			name: "http-no-compression",
			f: func() {
				p, err := httpclientUncompressed.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = httpclientUncompressed.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					panic(err)
				}
			},
		},
	}

	for _, bm := range benchmarks {
		bm := bm
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.f()
			}
		})
	}
	for n := 0; n < b.N; n++ {

	}
}
