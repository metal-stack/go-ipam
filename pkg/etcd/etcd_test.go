package etcd

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	ipam "github.com/metal-stack/go-ipam"
)

var (
	etcdOnce      sync.Once
	etcdContainer testcontainers.Container
	etcdVersion   string
)

func startEtcdForTest(ctx context.Context) (testcontainers.Container, *etcd, error) {
	etcdVersion = os.Getenv("ETCD_VERSION")
	if etcdVersion == "" {
		etcdVersion = "v3.7.0"
	}
	etcdOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "quay.io/coreos/etcd:" + etcdVersion,
			ExposedPorts: []string{"2379/tcp", "2380/tcp"},
			Cmd: []string{"etcd",
				"--name", "etcd",
				"--advertise-client-urls", "http://0.0.0.0:2379",
				"--initial-advertise-peer-urls", "http://0.0.0.0:2380",
				"--listen-client-urls", "http://0.0.0.0:2379",
				"--listen-peer-urls", "http://0.0.0.0:2380",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("ready to serve client requests"),
				wait.ForListeningPort("2379/tcp"),
				wait.ForListeningPort("2380/tcp"),
			),
		}
		etcdContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := etcdContainer.Host(ctx)
	if err != nil {
		return etcdContainer, nil, err
	}
	port, err := etcdContainer.MappedPort(ctx, "2379")
	if err != nil {
		return etcdContainer, nil, err
	}
	db, err := newEtcd(ctx, ip, port.Port(), nil, nil, true)
	if err != nil {
		return etcdContainer, nil, err
	}
	return etcdContainer, db, nil
}

func TestIntegrationEtcd(t *testing.T) {
	ctx := t.Context()
	_, storage, err := startEtcdForTest(ctx)
	require.NoError(t, err)

	ipamer := ipam.NewWithStorage(storage)

	// Tenant super network 1
	tenantSuper1, err := ipamer.NewPrefix(ctx, "10.64.0.0/14")
	require.NoError(t, err)

	require.NotNil(t, tenantSuper1)
	require.Equal(t, uint64(2), tenantSuper1.Usage().AcquiredIPs)
	require.Equal(t, uint64(65536), tenantSuper1.Usage().AvailableSmallestPrefixes)
	require.Equal(t, uint64(0), tenantSuper1.Usage().AcquiredPrefixes)

	cp, err := ipamer.AcquireChildPrefix(ctx, "10.64.0.0/14", 22)
	require.NoError(t, err)
	require.NotNil(t, cp)
	require.True(t, strings.HasPrefix(cp.Cidr, "10."))
	require.True(t, strings.HasSuffix(cp.Cidr, "/22"))
	require.Equal(t, "10.64.0.0/14", cp.ParentCidr)

	// reread
	tenantSuper1, err = ipamer.PrefixFrom(ctx, "10.64.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper1)
	require.Equal(t, uint64(1), tenantSuper1.Usage().AcquiredPrefixes)
	err = ipamer.ReleaseChildPrefix(ctx, cp)
	require.NoError(t, err)
	// reread
	tenantSuper1, err = ipamer.PrefixFrom(ctx, "10.64.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper1)
	require.Equal(t, uint64(0), tenantSuper1.Usage().AcquiredPrefixes)

	cp, err = ipamer.AcquireSpecificChildPrefix(ctx, "10.64.0.0/14", "10.64.0.0/22")
	require.NoError(t, err)
	require.NotNil(t, cp)
	require.Equal(t, "10.64.0.0/22", cp.String())
	require.Equal(t, "10.64.0.0/14", cp.ParentCidr)

	_, err = ipamer.AcquireIP(ctx, "10.64.0.0/14")
	require.EqualError(t, err, "prefix 10.64.0.0/14 has childprefixes, acquire ip not possible")

	// Tenant super network 2
	tenantSuper2, err := ipamer.NewPrefix(ctx, "10.76.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper2)
	require.Equal(t, uint64(2), tenantSuper2.Usage().AcquiredIPs)
	require.Equal(t, uint64(65536), tenantSuper2.Usage().AvailableSmallestPrefixes)
	require.Equal(t, uint64(0), tenantSuper2.Usage().AcquiredPrefixes)

	cp, err = ipamer.AcquireChildPrefix(ctx, "10.76.0.0/14", 22)
	require.NoError(t, err)
	require.NotNil(t, cp)
	require.True(t, strings.HasPrefix(cp.Cidr, "10."))
	require.True(t, strings.HasSuffix(cp.Cidr, "/22"))
	require.Equal(t, "10.76.0.0/14", cp.ParentCidr)

	// reread
	tenantSuper2, err = ipamer.PrefixFrom(ctx, "10.76.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper2)
	require.Equal(t, uint64(1), tenantSuper2.Usage().AcquiredPrefixes)
	err = ipamer.ReleaseChildPrefix(ctx, cp)
	require.NoError(t, err)
	// reread
	tenantSuper2, err = ipamer.PrefixFrom(ctx, "10.76.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper2)
	require.Equal(t, uint64(0), tenantSuper2.Usage().AcquiredPrefixes)

	cp, err = ipamer.AcquireSpecificChildPrefix(ctx, "10.76.0.0/14", "10.76.0.0/22")
	require.NoError(t, err)
	require.NotNil(t, cp)
	require.Equal(t, "10.76.0.0/22", cp.String())
	require.Equal(t, "10.76.0.0/14", cp.ParentCidr)

	// reread
	tenantSuper2, err = ipamer.PrefixFrom(ctx, "10.76.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper2)
	require.Equal(t, uint64(1), tenantSuper2.Usage().AcquiredPrefixes)
	err = ipamer.ReleaseChildPrefix(ctx, cp)
	require.NoError(t, err)
	// reread
	tenantSuper2, err = ipamer.PrefixFrom(ctx, "10.76.0.0/14")
	require.NoError(t, err)
	require.NotNil(t, tenantSuper2)
	require.Equal(t, uint64(0), tenantSuper2.Usage().AcquiredPrefixes)

	_, err = ipamer.AcquireIP(ctx, "10.76.0.0/14")
	require.EqualError(t, err, "prefix 10.76.0.0/14 has childprefixes, acquire ip not possible")

	// Read all child prefixes
	pfxs, err := storage.ReadAllPrefixes(ctx, ipam.DefaultNamespace)
	require.NoError(t, err)
	childPrefixesOfTenantSuper := make(map[string]bool)

	for _, pfx := range pfxs {
		if pfx.ParentCidr != "" {
			if pfx.ParentCidr != tenantSuper2.Cidr {
				continue
			}
			childPrefixesOfTenantSuper[pfx.String()] = false
		}
	}
	require.Len(t, childPrefixesOfTenantSuper, int(tenantSuper2.Usage().AcquiredPrefixes)) // nolint:gosec

	// Public Internet
	publicInternet, err := ipamer.NewPrefix(ctx, "1.2.3.0/25")
	require.NoError(t, err)
	require.NotNil(t, publicInternet)

	require.Equal(t, uint64(2), publicInternet.Usage().AcquiredIPs)
	require.Equal(t, uint64(128), publicInternet.Usage().AvailableIPs)
	require.Equal(t, "", publicInternet.ParentCidr)
	_, err = ipamer.AcquireChildPrefix(ctx, publicInternet.Cidr, 29)
	require.NoError(t, err)
	_, err = ipamer.AcquireSpecificChildPrefix(ctx, publicInternet.Cidr, "1.2.3.0/29")
	require.EqualError(t, err, "specific prefix 1.2.3.0/29 is not available in prefix 1.2.3.0/25")
	_, err = ipamer.AcquireIP(ctx, publicInternet.Cidr)
	require.EqualError(t, err, "prefix 1.2.3.0/25 has childprefixes, acquire ip not possible")
}
