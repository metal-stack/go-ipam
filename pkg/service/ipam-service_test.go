package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIpamService(t *testing.T) {
	t.Parallel()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		New(log, goipam.New(context.Background())),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	connectClient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)
	grpcClient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
		connect.WithGRPC(),
	)
	clients := []apiv1connect.IpamServiceClient{connectClient, grpcClient}

	t.Run("CreateDeleteGetPrefix", func(t *testing.T) {
		counter := 0
		for _, client := range clients {
			result, err := client.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, result.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.169.%d.0/24", counter))

			getresult, err := client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, getresult.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.169.%d.0/24", counter))

			deleteresult, err := client.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, deleteresult.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.169.%d.0/24", counter))

			_, err = client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			// FIXME with PrefixFrom returns error refactoring
			// nolint:testifylint
			require.Error(t, err, fmt.Errorf("prefix:'192.169.%d.0/24' not found", counter))

			counter++
		}
	})

	t.Run("AcquireReleaseChildPrefix", func(t *testing.T) {
		counter := 0
		for _, client := range clients {
			result, err := client.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
				Cidr: fmt.Sprintf("192.167.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, result.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.167.%d.0/24", counter))

			acquireresult, err := client.AcquireChildPrefix(context.Background(), connect.NewRequest(&v1.AcquireChildPrefixRequest{
				Cidr:   fmt.Sprintf("192.167.%d.0/24", counter),
				Length: 28,
			}))
			require.NoError(t, err)
			assert.Equal(t, acquireresult.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.167.%d.0/28", counter))

			releaseresult, err := client.ReleaseChildPrefix(context.Background(), connect.NewRequest(&v1.ReleaseChildPrefixRequest{
				Cidr: acquireresult.Msg.GetPrefix().GetCidr(),
			}))
			require.NoError(t, err)
			assert.Equal(t, releaseresult.Msg.GetPrefix().GetCidr(), acquireresult.Msg.GetPrefix().GetCidr())

			counter++
		}
	})

	t.Run("AcquireReleaseIP", func(t *testing.T) {
		counter := 0
		for _, client := range clients {
			result, err := client.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
				Cidr: fmt.Sprintf("192.166.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, result.Msg.GetPrefix().GetCidr(), fmt.Sprintf("192.166.%d.0/24", counter))

			acquireresult, err := client.AcquireIP(context.Background(), connect.NewRequest(&v1.AcquireIPRequest{
				PrefixCidr: fmt.Sprintf("192.166.%d.0/24", counter),
			}))
			require.NoError(t, err)
			assert.Equal(t, acquireresult.Msg.GetIp().GetIp(), fmt.Sprintf("192.166.%d.1", counter))

			releaseresult, err := client.ReleaseIP(context.Background(), connect.NewRequest(&v1.ReleaseIPRequest{
				PrefixCidr: fmt.Sprintf("192.166.%d.0/24", counter),
				Ip:         acquireresult.Msg.GetIp().GetIp(),
			}))
			require.NoError(t, err)
			assert.Equal(t, releaseresult.Msg.GetIp().GetIp(), fmt.Sprintf("192.166.%d.1", counter))

			counter++
		}
	})

	t.Run("CreateDeleteGetPrefixNamespaced", func(t *testing.T) {
		counter := 0
		for _, client := range clients {
			namespace := fmt.Sprintf("testns-%d", counter)
			_, err := client.CreateNamespace(context.Background(), connect.NewRequest(&v1.CreateNamespaceRequest{Namespace: namespace}))
			require.NoError(t, err)

			result, err := client.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
				Cidr:      "192.169.0.0/24",
				Namespace: &namespace,
			}))
			require.NoError(t, err)
			assert.Equal(t, "192.169.0.0/24", result.Msg.GetPrefix().GetCidr())

			getresult, err := client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr:      "192.169.0.0/24",
				Namespace: &namespace,
			}))
			require.NoError(t, err)
			assert.Equal(t, "192.169.0.0/24", getresult.Msg.GetPrefix().GetCidr())

			deleteresult, err := client.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
				Cidr:      "192.169.0.0/24",
				Namespace: &namespace,
			}))
			require.NoError(t, err)
			assert.Equal(t, "192.169.0.0/24", deleteresult.Msg.GetPrefix().GetCidr())

			_, err = client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr:      "192.169.0.0/24",
				Namespace: &namespace,
			}))
			require.Error(t, err, "prefix:'192.169.0.0/24' not found")
			counter++
		}
	})
}
