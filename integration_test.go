package ipam

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	_, storage, err := startPostgres()
	require.NoError(t, err)
	defer storage.db.Close()
	dump, err := ioutil.ReadFile("testdata/ipam.dump.sql")
	require.NoError(t, err)
	require.NotNil(t, dump)
	storage.db.MustExec(string(dump))
	ipam := NewWithStorage(storage)

	// Public Internet
	publicInternet := ipam.PrefixFrom("1.2.3.0/27")
	require.NotNil(t, publicInternet)

	require.Equal(t, uint64(25), publicInternet.Usage().AcquiredIPs)
	require.Equal(t, uint64(32), publicInternet.Usage().AvailableIPs)
	require.Equal(t, "", publicInternet.ParentCidr)
	_, err = ipam.AcquireChildPrefix(publicInternet.Cidr, 29)
	require.EqualError(t, err, "prefix 1.2.3.0/27 has ips, acquire child prefix not possible")
	ip, err := ipam.AcquireIP(publicInternet.Cidr)
	require.NoError(t, err)
	require.NotNil(t, ip)
	require.True(t, strings.HasPrefix(ip.IP.String(), "1.2.3"))
	require.Equal(t, "1.2.3.20", ip.IP.String())
	// reread prefix
	publicInternet = ipam.PrefixFrom("1.2.3.0/27")
	require.Equal(t, uint64(26), publicInternet.Usage().AcquiredIPs)
	_, err = ipam.ReleaseIP(ip)
	require.NoError(t, err)
	// reread prefix
	publicInternet = ipam.PrefixFrom("1.2.3.0/27")
	require.Equal(t, uint64(25), publicInternet.Usage().AcquiredIPs)

	// Tenant super network
	tenantSuper := ipam.PrefixFrom("10.128.0.0/14")
	require.NotNil(t, tenantSuper)
	require.Equal(t, uint64(2), tenantSuper.Usage().AcquiredIPs)
	// FIXME validate why 60928
	require.Equal(t, 60928, int(tenantSuper.Usage().AvailableSmallestPrefixes))
	// FIXME This Super Prefix has leaked child prefixes !
	require.Equal(t, uint64(18), tenantSuper.Usage().AcquiredPrefixes)

	cp, err := ipam.AcquireChildPrefix("10.128.0.0/14", 22)
	require.NoError(t, err)
	require.NotNil(t, cp)
	require.True(t, strings.HasPrefix(cp.Cidr, "10."))
	require.True(t, strings.HasSuffix(cp.Cidr, "/22"))
	require.Equal(t, "10.128.0.0/14", cp.ParentCidr)

	// reread
	tenantSuper = ipam.PrefixFrom("10.128.0.0/14")
	require.NotNil(t, tenantSuper)
	require.Equal(t, uint64(19), tenantSuper.Usage().AcquiredPrefixes)
	err = ipam.ReleaseChildPrefix(cp)
	require.NoError(t, err)
	// reread
	tenantSuper = ipam.PrefixFrom("10.128.0.0/14")
	require.NotNil(t, tenantSuper)
	require.Equal(t, uint64(18), tenantSuper.Usage().AcquiredPrefixes)

	_, err = ipam.AcquireIP("10.128.0.0/14")
	require.EqualError(t, err, "prefix 10.128.0.0/14 has childprefixes, acquire ip not possible")

	// Read all child prefixes
	pfxs, err := storage.ReadAllPrefixes()
	require.NoError(t, err)
	childPrefixCount := 0
	for _, pfx := range pfxs {
		if pfx.ParentCidr != "" {
			require.Equal(t, tenantSuper.Cidr, pfx.ParentCidr)
			childPrefixCount++
		}
	}
	// FIXME This Super Prefix has leaked child prefixes !
	// require.Equal(t, childPrefixCount, tenantSuper.Usage().AcquiredPrefixes)
}