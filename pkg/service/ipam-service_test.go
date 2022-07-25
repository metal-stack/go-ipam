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

	t.Run("createDeleteGetPrefix", func(t *testing.T) {
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

			getresult, err = client.GetPrefix(context.Background(), connect.NewRequest(&v1.GetPrefixRequest{
				Cidr: fmt.Sprintf("192.169.%d.0/24", counter),
			}))
			require.Error(t, err, fmt.Errorf("prefix:'192.169.%d.0/24' not found", counter))

			counter++
		}
	})
}
