package test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	compress "github.com/klauspost/connect-compress/v2"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"github.com/stretchr/testify/require"
)

// BenchmarkGrpcImpact located in a separate package to prevent import cycles.
func BenchmarkGrpcImpact(b *testing.B) {
	ctx := b.Context()
	ipam := goipam.New(ctx)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(log, ipam),
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
		f    func() error
	}{
		{
			name: "library",
			f: func() error {
				p, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
				if err != nil {
					return err
				}
				if p == nil {
					return fmt.Errorf("Prefix nil:%w", err)
				}
				_, err = ipam.DeletePrefix(ctx, p.Cidr)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "grpc",
			f: func() error {
				p, err := grpc.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				if p == nil {
					return fmt.Errorf("Prefix nil:%w", err)
				}
				_, err = grpc.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "grpc-no-compression",
			f: func() error {
				p, err := grpcUncompressed.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				if p == nil {
					return fmt.Errorf("Prefix nil:%w", err)
				}
				_, err = grpcUncompressed.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "http",
			f: func() error {
				p, err := httpclient.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				if p == nil {
					return fmt.Errorf("Prefix nil:%w", err)
				}
				_, err = httpclient.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "http-no-compression",
			f: func() error {
				p, err := httpclientUncompressed.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				if p == nil {
					return fmt.Errorf("Prefix nil:%w", err)
				}
				_, err = httpclientUncompressed.DeletePrefix(ctx, connect.NewRequest(&v1.DeletePrefixRequest{
					Cidr: "192.169.0.0/24",
				}))
				if err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				err := bm.f()
				require.NoError(b, err)
			}
		})
	}
}
