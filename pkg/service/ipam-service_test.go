package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bufbuild/connect-go"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestIpamService(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		New(zaptest.NewLogger(t).Sugar(), goipam.New()),
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
			require.Nil(t, err)
			assert.Equal(t, result.Msg.Prefix.Cidr, fmt.Sprintf("192.169.%d.0/24", counter))

			getresult, err := client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.Nil(t, err)
			assert.Equal(t, getresult.Msg.Prefix.Cidr, fmt.Sprintf("192.169.%d.0/24", counter))

			deleteresult, err := client.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.Nil(t, err)
			assert.Equal(t, deleteresult.Msg.Prefix.Cidr, fmt.Sprintf("192.169.%d.0/24", counter))

			_, err = client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
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
			require.Nil(t, err)
			assert.Equal(t, result.Msg.Prefix.Cidr, fmt.Sprintf("192.167.%d.0/24", counter))

			acquireresult, err := client.AcquireChildPrefix(context.Background(), connect.NewRequest(&v1.AcquireChildPrefixRequest{
				Cidr:   fmt.Sprintf("192.167.%d.0/24", counter),
				Length: 28,
			}))
			require.Nil(t, err)
			assert.Equal(t, acquireresult.Msg.Prefix.Cidr, fmt.Sprintf("192.167.%d.0/28", counter))

			releaseresult, err := client.ReleaseChildPrefix(context.Background(), connect.NewRequest(&v1.ReleaseChildPrefixRequest{
				Cidr: acquireresult.Msg.Prefix.Cidr,
			}))
			require.Nil(t, err)
			assert.Equal(t, releaseresult.Msg.Prefix.Cidr, acquireresult.Msg.Prefix.Cidr)

			counter++
		}
	})

	t.Run("AcquireReleaseIP", func(t *testing.T) {
		counter := 0
		for _, client := range clients {
			result, err := client.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
				Cidr: fmt.Sprintf("192.166.%d.0/24", counter),
			}))
			require.Nil(t, err)
			assert.Equal(t, result.Msg.Prefix.Cidr, fmt.Sprintf("192.166.%d.0/24", counter))

			acquireresult, err := client.AcquireIP(context.Background(), connect.NewRequest(&v1.AcquireIPRequest{
				PrefixCidr: fmt.Sprintf("192.166.%d.0/24", counter),
			}))
			require.Nil(t, err)
			assert.Equal(t, acquireresult.Msg.Ip.Ip, fmt.Sprintf("192.166.%d.1", counter))

			releaseresult, err := client.ReleaseIP(context.Background(), connect.NewRequest(&v1.ReleaseIPRequest{
				PrefixCidr: fmt.Sprintf("192.166.%d.0/24", counter),
				Ip:         acquireresult.Msg.Ip.Ip,
			}))
			require.Nil(t, err)
			assert.Equal(t, releaseresult.Msg.Ip.Ip, fmt.Sprintf("192.166.%d.1", counter))

			counter++
		}
	})
}
