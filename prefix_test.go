package ipam

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// getHostAddresses will return all possible ipadresses a host can get in the given prefix.
// The IPs will be acquired by this method, so that the prefix has no free IPs afterwards.
func (i *ipamer) getHostAddresses(ctx context.Context, prefix string) ([]string, error) {
	hostAddresses := []string{}

	p, err := i.NewPrefix(ctx, prefix)
	if err != nil {
		return hostAddresses, err
	}

	// loop till AcquireIP signals that it has no ips left
	for {
		ip, err := i.AcquireIP(ctx, p.Cidr)
		if errors.Is(err, ErrNoIPAvailable) {
			return hostAddresses, nil
		}
		if err != nil {
			return nil, err
		}
		hostAddresses = append(hostAddresses, ip.IP.String())
	}
}

func TestIPRangeOverlapping(t *testing.T) {
	ctx := context.Background()
	i := New(ctx)

	cidr := "10.10.10.0/24"
	_, err := i.NewPrefix(ctx, cidr)
	require.NoError(t, err)

	cidr = "10.10.10.1/24"
	_, err = i.NewPrefix(ctx, cidr)
	require.Error(t, err)
}

func TestIpamer_AcquireIP(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		prefixCIDR  string
		namespace   string
		existingips []string
	}
	tests := []struct {
		name   string
		fields fields
		want   *IP
	}{
		{
			name: "Acquire next IP regularly",
			fields: fields{
				prefixCIDR:  "192.168.1.0/24",
				existingips: []string{},
			},
			want: &IP{IP: netip.MustParseAddr("192.168.1.1"), ParentPrefix: "192.168.1.0/24"},
		},
		{
			name: "Acquire next namespaced IP regularly",
			fields: fields{
				prefixCIDR:  "192.168.1.0/24",
				namespace:   "my-namespace",
				existingips: []string{},
			},
			want: &IP{IP: netip.MustParseAddr("192.168.1.1"), ParentPrefix: "192.168.1.0/24"},
		},
		{
			name: "Acquire next IPv6 regularly",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/124",
				existingips: []string{},
			},
			want: &IP{IP: netip.MustParseAddr("2001:0db8:85a3::1"), ParentPrefix: "2001:0db8:85a3::/124"},
		},
		{
			name: "Acquire next namespaced IPv6 regularly",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/124",
				namespace:   "my-namespace",
				existingips: []string{},
			},
			want: &IP{IP: netip.MustParseAddr("2001:0db8:85a3::1"), ParentPrefix: "2001:0db8:85a3::/124"},
		},
		{
			name: "Want next IP, network already occupied a little",
			fields: fields{
				prefixCIDR:  "192.168.2.0/30",
				existingips: []string{"192.168.2.1"},
			},
			want: &IP{IP: netip.MustParseAddr("192.168.2.2"), ParentPrefix: "192.168.2.0/30"},
		},
		{
			name: "Want next IPv6, network already occupied a little",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/124",
				existingips: []string{"2001:db8:85a3::1"},
			},
			want: &IP{IP: netip.MustParseAddr("2001:db8:85a3::2"), ParentPrefix: "2001:0db8:85a3::/124"},
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				prefixCIDR:  "192.168.3.0/30",
				existingips: []string{"192.168.3.1", "192.168.3.2"},
			},
			want: nil,
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				prefixCIDR: "192.168.4.0/32",
			},
			want: nil,
		},
		{
			name: "Want next IPv6, but network is full",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/126",
				existingips: []string{"2001:db8:85a3::1", "2001:db8:85a3::2", "2001:db8:85a3::3"},
			},
			want: nil,
		},
		{
			name: "Want next IPv6, but network is full",
			fields: fields{
				prefixCIDR: "2001:0db8:85a3::/128",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		test := tt
		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			p, err := ipam.NewPrefix(ctx, test.fields.prefixCIDR)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			t.Logf("Prefix:%#v", p)
			for _, ipString := range test.fields.existingips {
				p.ips[ipString] = true
			}

			var updatedPrefix Prefix
			updatedPrefix, err = ipam.storage.UpdatePrefix(ctx, *p, defaultNamespace)
			if err != nil {
				t.Errorf("Could not update prefix: %v", err)
			}
			got, _ := ipam.AcquireIP(ctx, updatedPrefix.Cidr)
			if test.want == nil || got == nil {
				if !reflect.DeepEqual(got, test.want) {
					t.Errorf("Ipamer.AcquireIP() want or got is nil, got %v, want %v", got, test.want)
				}
			} else {
				if test.want.IP.Compare(got.IP) != 0 {
					t.Errorf("Ipamer.AcquireIP() got %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestIpamer_ReleaseIPFromPrefixIPv4(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		err = ipam.ReleaseIPFromPrefix(ctx, prefix.Cidr, "1.2.3.4")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, "NotFound: unable to release ip:1.2.3.4 because it is not allocated in prefix:192.168.0.0/24", err.Error())

		err = ipam.ReleaseIPFromPrefix(ctx, "4.5.6.7/23", "1.2.3.4")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr:4.5.6.7/23")
	})
}

func TestIpamer_ReleaseIPFromPrefixIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:0db8:85a3::/120")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		err = ipam.ReleaseIPFromPrefix(ctx, prefix.Cidr, "1.2.3.4")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, "NotFound: unable to release ip:1.2.3.4 because it is not allocated in prefix:2001:db8:85a3::/120", err.Error())

		err = ipam.ReleaseIPFromPrefix(ctx, prefix.Cidr, "1001:0db8:85a3::1")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, "NotFound: unable to release ip:1001:0db8:85a3::1 because it is not allocated in prefix:2001:db8:85a3::/120", err.Error())

		err = ipam.ReleaseIPFromPrefix(ctx, "1001:0db8:85a3::/120", "1.2.3.4")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr:1001:0db8:85a3::/120")
	})
}
func TestIpamer_AcquireSpecificIP(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		// IPv4
		prefix, err := ipam.NewPrefix(ctx, "192.168.99.0/24")
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		// network an broadcast are blocked
		require.Equal(t, uint64(2), prefix.acquiredips())
		ip1, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.99.1")
		require.NoError(t, err)
		require.NotNil(t, ip1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())
		ip2, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.99.2")
		require.NoError(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(4), prefix.acquiredips())
		require.Equal(t, "192.168.99.1", ip1.IP.String())
		require.Equal(t, "192.168.99.2", ip2.IP.String())

		// Wish IP out of prefix
		ip3, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.98.2")
		require.Nil(t, ip3)
		require.Error(t, err)
		require.Equal(t, "given ip:192.168.98.2 is not in 192.168.99.0/24", err.Error())

		// Wish IP is invalid
		ip4, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.99.1.invalid")
		require.Nil(t, ip4)
		require.Error(t, err)
		require.Equal(t, "given ip:192.168.99.1.invalid in not valid", err.Error())

		// Cidr is invalid
		ip5, err := ipam.AcquireSpecificIP(ctx, "3.4.5.6/27", "192.168.99.1.invalid")
		require.Nil(t, ip5)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr:3.4.5.6/27")

		prefix, err = ipam.ReleaseIP(ctx, ip1)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())

		prefix, err = ipam.ReleaseIP(ctx, ip2)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())

		// IPv6
		prefix, err = ipam.NewPrefix(ctx, "2001:0db8:85a3::/120")
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		// network is blocked
		require.Equal(t, uint64(1), prefix.acquiredips())
		ip1, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "2001:db8:85a3::1")
		require.NoError(t, err)
		require.NotNil(t, ip1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())
		ip2, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "2001:0db8:85a3::2")
		require.NoError(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())
		require.Equal(t, "2001:db8:85a3::1", ip1.IP.String())
		require.Equal(t, "2001:db8:85a3::2", ip2.IP.String())

		// Wish IP out of prefix
		ip3, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "2001:0db8:85a4::1")
		require.Nil(t, ip3)
		require.Error(t, err)
		require.Equal(t, "given ip:2001:0db8:85a4::1 is not in 2001:db8:85a3::/120", err.Error())

		// Cidr is invalid
		ip5, err = ipam.AcquireSpecificIP(ctx, "2001:0db8:95a3::/120", "2001:0db8:95a3::invalid")
		require.Nil(t, ip5)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr:2001:0db8:95a3::/120", err.Error())

		prefix, err = ipam.ReleaseIP(ctx, ip1)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())

		prefix, err = ipam.ReleaseIP(ctx, ip2)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(1), prefix.acquiredips())
	})
}

func TestIpamer_AcquireIPCountsIPv4(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		// network and broadcast are blocked
		require.Equal(t, uint64(2), prefix.acquiredips())
		ip1, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())
		ip2, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(4), prefix.acquiredips())
		require.True(t, strings.HasPrefix(ip1.IP.String(), "192.168.0"))
		require.True(t, strings.HasPrefix(ip2.IP.String(), "192.168.0"))

		prefix, err = ipam.ReleaseIP(ctx, ip1)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())

		prefix, err = ipam.ReleaseIP(ctx, ip2)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())
	})
}

func TestIpamer_AcquireIPCountsIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:0db8:85a3::/120")
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		// network is blocked
		require.Equal(t, uint64(1), prefix.acquiredips())
		ip1, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())
		ip2, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(3), prefix.acquiredips())
		require.True(t, strings.HasPrefix(ip1.IP.String(), "2001:db8:85a3::"))
		require.True(t, strings.HasPrefix(ip2.IP.String(), "2001:db8:85a3::"))

		prefix, err = ipam.ReleaseIP(ctx, ip1)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(2), prefix.acquiredips())

		prefix, err = ipam.ReleaseIP(ctx, ip2)
		require.NoError(t, err)
		require.Equal(t, uint64(256), prefix.availableips())
		require.Equal(t, uint64(1), prefix.acquiredips())
	})
}

func TestIpamer_AcquireChildPrefixFragmented(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		allPrefixes, err := ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Empty(t, allPrefixes)

		// Create Prefix with /20
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, 1024, int(s))
		require.Empty(t, prefix.acquiredPrefixes())
		require.Empty(t, prefix.Usage().AcquiredPrefixes)

		// Acquire first half 192.168.0.0/21 = 192.168.0.0 - 192.168.7.254
		c1, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.NoError(t, err)
		require.NotNil(t, c1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, 512, int(s))
		require.Equal(t, 1, int(prefix.acquiredPrefixes()))
		require.Equal(t, 1, int(prefix.Usage().AcquiredPrefixes))

		// acquire 1/4the of the rest 192.168.8.0/22 = 192.168.8.0 - 192.168.11.254
		c2, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, a := prefix.availablePrefixes()
		// Next free must be 192.168.12.0/22
		require.Equal(t, []string{"192.168.12.0/22"}, a)
		require.Equal(t, 256, int(s))
		require.Equal(t, 2, int(prefix.acquiredPrefixes()))
		require.Equal(t, 2, int(prefix.Usage().AcquiredPrefixes))

		// acquire impossible size
		_, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.EqualError(t, err, "no prefix found in 192.168.0.0/20 with length:21, but 192.168.12.0/22 is available")

		// Release small, first half acquired
		err = ipam.ReleaseChildPrefix(ctx, c2)
		require.NoError(t, err)

		// acquire /28
		c3, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 28)
		require.NoError(t, err)
		require.NotNil(t, c3)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, a = prefix.availablePrefixes()
		require.Equal(t, []string{"192.168.8.16/28", "192.168.8.32/27", "192.168.8.64/26", "192.168.8.128/25", "192.168.9.0/24", "192.168.10.0/23", "192.168.12.0/22"}, a)
		require.Equal(t, 508, int(s))
		require.Equal(t, 2, int(prefix.acquiredPrefixes()))
		require.Equal(t, 2, int(prefix.Usage().AcquiredPrefixes))

		// acquire impossible size
		_, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.EqualError(t, err, "no prefix found in 192.168.0.0/20 with length:21, but 192.168.8.16/28,192.168.8.32/27,192.168.8.64/26,192.168.8.128/25,192.168.9.0/24,192.168.10.0/23,192.168.12.0/22 are available")

		// acquire a /22 which must be possible
		c4, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c4)
		require.Equal(t, "192.168.12.0/22", c4.String())

	})
}

func TestIpamer_AcquireChildPrefixCounts(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		allPrefixes, err := ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Empty(t, allPrefixes)

		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, uint64(0), prefix.Usage().AcquiredPrefixes)

		usage := prefix.Usage()
		require.Equal(t, "ip:2/4096", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 1)

		c1, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c1)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(768), s)
		require.Equal(t, uint64(1), prefix.acquiredPrefixes())
		require.Equal(t, uint64(1), prefix.Usage().AcquiredPrefixes)

		usage = prefix.Usage()
		require.Equal(t, "ip:2/4096 prefixes alloc:1 avail:768", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 2)

		c2, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c2)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(512), s)
		require.Equal(t, uint64(2), prefix.acquiredPrefixes())
		require.Equal(t, uint64(2), prefix.Usage().AcquiredPrefixes)
		require.True(t, strings.HasSuffix(c1.Cidr, "/22"))
		require.True(t, strings.HasSuffix(c2.Cidr, "/22"))
		require.True(t, strings.HasPrefix(c1.Cidr, "192.168."))
		require.True(t, strings.HasPrefix(c2.Cidr, "192.168."))
		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 3)

		err = ipam.ReleaseChildPrefix(ctx, c1)
		require.NoError(t, err)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(768), s)
		require.Equal(t, uint64(1), prefix.acquiredPrefixes())
		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 2)

		err = ipam.ReleaseChildPrefix(ctx, c2)
		require.NoError(t, err)
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, uint64(0), prefix.Usage().AcquiredPrefixes)
		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 1)

		err = ipam.ReleaseChildPrefix(ctx, c1)
		require.Errorf(t, err, "unable to release prefix %s:delete prefix:%s not found", prefix.Cidr, c1.Cidr)

		c3, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c2)
		ip1, err := ipam.AcquireIP(ctx, c3.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip1)

		err = ipam.ReleaseChildPrefix(ctx, c3)
		require.Errorf(t, err, "prefix %s has ips, deletion not possible", c3.Cidr)

		c3, err = ipam.ReleaseIP(ctx, ip1)
		require.NoError(t, err)

		err = ipam.ReleaseChildPrefix(ctx, c3)
		require.NoError(t, err)

		allPrefixes, err = ipam.storage.ReadAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		require.Len(t, allPrefixes, 1)
	})
}

func TestIpamer_AcquireChildPrefixIPv4(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		// Same length
		cp, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 20)
		require.Error(t, err)
		require.Equal(t, "given length:20 must be greater than prefix length:20", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.NoError(t, err)
		require.NotNil(t, cp)
		require.True(t, strings.HasPrefix(cp.Cidr, "192.168."))
		require.True(t, strings.HasSuffix(cp.Cidr, "/21"))
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// No more ChildPrefixes
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.NoError(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 21)
		require.Error(t, err)
		require.Equal(t, "no prefix found in 192.168.0.0/20 with length:21", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix(ctx, "10.0.0.0/24")
		require.NoError(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p2.acquiredPrefixes())
		ip, err := ipam.AcquireIP(ctx, p2.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireChildPrefix(ctx, p2.Cidr, 25)
		require.Error(t, err)
		require.Equal(t, "prefix 10.0.0.0/24 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Children, AcquireIP wont work
		p3, err := ipam.NewPrefix(ctx, "172.17.0.0/24")
		require.NoError(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p3.acquiredPrefixes())
		cp3, err := ipam.AcquireChildPrefix(ctx, p3.Cidr, 25)
		require.NoError(t, err)
		require.NotNil(t, cp3)
		p3, err = ipam.PrefixFrom(ctx, p3.Cidr)
		require.NoError(t, err)
		ip, err = ipam.AcquireIP(ctx, p3.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(ctx, p3)
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr")
	})
}

func TestIpamer_AcquireChildPrefixIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:0db8:85a3::/116")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		// Same length
		cp, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 116)
		require.Error(t, err)
		require.Equal(t, "given length:116 must be greater than prefix length:116", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 117)
		require.NoError(t, err)
		require.NotNil(t, cp)
		require.True(t, strings.HasPrefix(cp.Cidr, "2001:db8:85a3:"))
		require.True(t, strings.HasSuffix(cp.Cidr, "/117"))
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// No more ChildPrefixes
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 117)
		require.NoError(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireChildPrefix(ctx, prefix.Cidr, 117)
		require.Error(t, err)
		require.Equal(t, "no prefix found in 2001:db8:85a3::/116 with length:117", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix(ctx, "2001:0db8:95a3::/120")
		require.NoError(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p2.acquiredPrefixes())
		ip, err := ipam.AcquireIP(ctx, p2.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireChildPrefix(ctx, p2.Cidr, 121)
		require.Error(t, err)
		require.Equal(t, "prefix 2001:db8:95a3::/120 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Children, AcquireIP wont work
		p3, err := ipam.NewPrefix(ctx, "2001:0db8:75a3::/120")
		require.NoError(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p3.acquiredPrefixes())
		cp3, err := ipam.AcquireChildPrefix(ctx, p3.Cidr, 121)
		require.NoError(t, err)
		require.NotNil(t, cp3)
		p3, err = ipam.PrefixFrom(ctx, p3.Cidr)
		require.NoError(t, err)
		ip, err = ipam.AcquireIP(ctx, p3.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 2001:db8:75a3::/120 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(ctx, p3)
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr")
	})
}

func TestIpamer_AcquireSpecificChildPrefixIPv4(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		// Same length
		cp, err := ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "192.168.0.0/20")
		require.Error(t, err)
		require.Equal(t, "given length:20 must be greater than prefix length:20", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "192.168.0.0/21")
		require.NoError(t, err)
		require.NotNil(t, cp)
		require.Equal(t, "192.168.0.0/21", cp.Cidr)
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// specific prefix not available
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "192.168.8.0/21")
		require.NoError(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "192.168.8.0/21")
		require.Error(t, err)
		require.Equal(t, "specific prefix 192.168.8.0/21 is not available in prefix 192.168.0.0/20", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix(ctx, "10.0.0.0/24")
		require.NoError(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p2.acquiredPrefixes())
		ip, err := ipam.AcquireIP(ctx, p2.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireSpecificChildPrefix(ctx, p2.Cidr, "10.0.0.0/25")
		require.Error(t, err)
		require.Equal(t, "prefix 10.0.0.0/24 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Children, AcquireIP wont work
		p3, err := ipam.NewPrefix(ctx, "172.17.0.0/24")
		require.NoError(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p3.acquiredPrefixes())
		cp3, err := ipam.AcquireSpecificChildPrefix(ctx, p3.Cidr, "172.17.0.0/25")
		require.NoError(t, err)
		require.NotNil(t, cp3)
		p3, err = ipam.PrefixFrom(ctx, p3.Cidr)
		require.NoError(t, err)
		ip, err = ipam.AcquireIP(ctx, p3.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)
	})
}

func TestIpamer_AcquireSpecificChildPrefixIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:0db8:85a3::/116")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		// Same length
		cp, err := ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "2001:0db8:85a3::/116")
		require.Error(t, err)
		require.Equal(t, "given length:116 must be greater than prefix length:116", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "2001:0db8:85a3::/117")
		require.NoError(t, err)
		require.NotNil(t, cp)
		require.Equal(t, "2001:db8:85a3::/117", cp.Cidr)
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// specific prefix not available
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "2001:0db8:85a3::0800/117")
		require.NoError(t, err)
		require.NotNil(t, cp)
		require.Equal(t, "2001:db8:85a3::800/117", cp.Cidr)
		cp, err = ipam.AcquireSpecificChildPrefix(ctx, prefix.Cidr, "2001:0db8:85a3::0800/117")
		require.Error(t, err)
		require.Equal(t, "specific prefix 2001:0db8:85a3::0800/117 is not available in prefix 2001:db8:85a3::/116", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix(ctx, "2001:0db8:95a3::/120")
		require.NoError(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p2.acquiredPrefixes())
		ip, err := ipam.AcquireIP(ctx, p2.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireSpecificChildPrefix(ctx, p2.Cidr, "2001:0db8:95a3::/121")
		require.Error(t, err)
		require.Equal(t, "prefix 2001:db8:95a3::/120 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Children, AcquireIP wont work
		p3, err := ipam.NewPrefix(ctx, "2001:0db8:75a3::/120")
		require.NoError(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), p3.acquiredPrefixes())
		cp3, err := ipam.AcquireSpecificChildPrefix(ctx, p3.Cidr, "2001:0db8:75a3::/121")
		require.NoError(t, err)
		require.NotNil(t, cp3)
		p3, err = ipam.PrefixFrom(ctx, p3.Cidr)
		require.NoError(t, err)
		ip, err = ipam.AcquireIP(ctx, p3.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 2001:db8:75a3::/120 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(ctx, p3)
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
		require.Contains(t, err.Error(), "NotFound: unable to find prefix for cidr")
	})
}

func TestIpamer_AcquireChildPrefixNoDuplicatesUntilFullIPv6(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:0db8:85a3::/112")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(16384), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		uniquePrefixes := make(map[string]bool)
		// acquire all /120 prefixes (2^8 = 256)
		for range 256 {
			cp, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 120)
			require.NoError(t, err)
			require.NotNil(t, cp)
			require.True(t, strings.HasPrefix(cp.Cidr, "2001:db8:85a3:"))
			require.True(t, strings.HasSuffix(cp.Cidr, "/120"))
			require.Equal(t, prefix.Cidr, cp.ParentCidr)
			_, ok := uniquePrefixes[cp.String()]
			require.False(t, ok)
			uniquePrefixes[cp.String()] = true
		}
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Len(t, uniquePrefixes, 256)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(0), s)
		require.Equal(t, uint64(256), prefix.acquiredPrefixes())

	})
}
func TestIpamer_AcquireChildPrefixNoDuplicatesUntilFullIPv4(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/16")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(16384), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		uniquePrefixes := make(map[string]bool)
		// acquire all /24 prefixes (2^8 = 256)
		for range 256 {
			cp, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 24)
			require.NoError(t, err)
			require.NotNil(t, cp)
			require.True(t, strings.HasPrefix(cp.Cidr, "192.168."))
			require.True(t, strings.HasSuffix(cp.Cidr, "/24"))
			require.Equal(t, prefix.Cidr, cp.ParentCidr)
			_, ok := uniquePrefixes[cp.String()]
			require.False(t, ok)
			uniquePrefixes[cp.String()] = true
		}
		prefix, err = ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.Len(t, uniquePrefixes, 256)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(0), s)
		require.Equal(t, uint64(256), prefix.acquiredPrefixes())

	})
}

func TestPrefix_Availableips(t *testing.T) {

	tests := []struct {
		name string
		Cidr string
		want uint64
	}{
		{
			name: "large",
			Cidr: "192.168.0.0/20",
			want: 4096,
		},
		{
			name: "small",
			Cidr: "192.168.0.0/24",
			want: 256,
		},
		{
			name: "smaller",
			Cidr: "192.168.0.0/25",
			want: 128,
		},
		{
			name: "smaller",
			Cidr: "192.168.0.0/30",
			want: 4,
		},
		{
			name: "smaller IPv6",
			Cidr: "2001:0db8:85a3::/126",
			want: 4,
		},
		{
			name: "large IPv6",
			Cidr: "2001:0db8:85a3::/116",
			want: 4096,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr: test.Cidr,
			}
			if got := p.availableips(); got != test.want {
				t.Errorf("Prefix.Availableips() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIpamer_PrefixesOverlapping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		existingPrefixes []string
		newPrefixes      []string
		wantErr          bool
		errorString      string
	}{
		{
			name:             "simple",
			existingPrefixes: []string{"192.168.0.0/24"},
			newPrefixes:      []string{"192.168.1.0/24"},
			wantErr:          false,
			errorString:      "",
		},
		{
			name:             "simple IPv6",
			existingPrefixes: []string{"2001:0db8:85a3::/126"},
			newPrefixes:      []string{"2001:0db8:85a4::/126"},
			wantErr:          false,
			errorString:      "",
		},
		{
			name:             "one overlap IPv6",
			existingPrefixes: []string{"2001:0db8:85a3::/126", "2001:0db8:85a4::/126"},
			newPrefixes:      []string{"2001:0db8:85a4::/126"},
			wantErr:          true,
			errorString:      "2001:db8:85a4::/126 overlaps 2001:db8:85a4::/126",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/24", "192.168.1.0/24"},
			newPrefixes:      []string{"192.168.1.0/24"},
			wantErr:          true,
			errorString:      "192.168.1.0/24 overlaps 192.168.1.0/24",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/24", "192.168.1.0/24"},
			newPrefixes:      []string{"192.168.0.0/23"},
			wantErr:          true,
			errorString:      "192.168.0.0/23 overlaps 192.168.0.0/24",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/23", "192.168.2.0/23"},
			newPrefixes:      []string{"192.168.3.0/24"},
			wantErr:          true,
			errorString:      "192.168.3.0/24 overlaps 192.168.2.0/23",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.128.0/25"},
			newPrefixes:      []string{"192.168.128.0/27"},
			wantErr:          true,
			errorString:      "192.168.128.0/27 overlaps 192.168.128.0/25",
		},
	}
	for _, tt := range tests {
		test := tt
		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			for _, ep := range test.existingPrefixes {
				p, err := ipam.NewPrefix(ctx, ep)
				if err != nil {
					t.Errorf("Newprefix on ExistingPrefix failed:%v", err)
				}
				if p == nil {
					t.Errorf("Newprefix on ExistingPrefix returns nil")
				}
			}
			err := PrefixesOverlapping(test.existingPrefixes, test.newPrefixes)
			if test.wantErr && err == nil {
				t.Errorf("Ipamer.PrefixesOverlapping() expected error but err was nil")
			}
			if test.wantErr && err != nil && err.Error() != test.errorString {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v, errorString = %v", err, test.wantErr, test.errorString)
			}
			if !test.wantErr && err != nil {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestIpamer_NewPrefix(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		cidr        string
		wantcidr    string
		wantErr     bool
		errorString string
	}{
		{
			name:     "valid Prefix",
			cidr:     "192.168.0.0/24",
			wantcidr: "192.168.0.0/24",
			wantErr:  false,
		},
		{
			name:     "valid Prefix, not in canonical form",
			cidr:     "192.169.0.1/24",
			wantcidr: "192.169.0.0/24",
			wantErr:  false,
		},
		{
			name:     "valid Prefix, not in canonical form",
			cidr:     "192.167.10.0/16",
			wantcidr: "192.167.0.0/16",
			wantErr:  false,
		},
		{
			name:        "invalid Prefix",
			cidr:        "192.168.0.0/33",
			wantErr:     true,
			errorString: "unable to parse cidr:192.168.0.0/33 netip.ParsePrefix(\"192.168.0.0/33\"): prefix length out of range",
		},
		{
			name:     "valid IPv6 Prefix",
			cidr:     "2001:0db8:85a3::/120",
			wantcidr: "2001:db8:85a3::/120",
			wantErr:  false,
		},
		{
			name:     "valid IPv6 Prefix, not in canonical form",
			cidr:     "2001:0db8:85a4::2/120",
			wantcidr: "2001:db8:85a4::/120",
			wantErr:  false,
		},
		{
			name:        "invalid IPv6 Prefix length",
			cidr:        "2001:0db8:85a3::/129",
			wantErr:     true,
			errorString: "unable to parse cidr:2001:0db8:85a3::/129 netip.ParsePrefix(\"2001:0db8:85a3::/129\"): prefix length out of range",
		},
		{
			name:        "invalid IPv6 Prefix length",
			cidr:        "2001:0db8:85a3:::/120",
			wantErr:     true,
			errorString: "unable to parse cidr:2001:0db8:85a3:::/120 netip.ParsePrefix(\"2001:0db8:85a3:::/120\"): ParseAddr(\"2001:0db8:85a3:::\"): each colon-separated field must have at least one digit (at \":\")",
		},
	}
	for _, tt := range tests {
		test := tt
		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			got, err := ipam.NewPrefix(ctx, test.cidr)
			if (err != nil) != test.wantErr {
				t.Errorf("Ipamer.NewPrefix(ctx, ) error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if (err != nil) && test.errorString != err.Error() {
				t.Errorf("Ipamer.NewPrefix(ctx, ) error = %v, errorString %v", err, test.errorString)
				return
			}

			if err != nil {
				return
			}
			if got.Cidr != test.wantcidr {
				t.Errorf("Ipamer.NewPrefix(ctx, ) cidr = %v, want %v", got.Cidr, test.wantcidr)
			}
		})
	}
}

func TestIpamer_DeletePrefix(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		// IPv4
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, uint64(0), prefix.Usage().AcquiredPrefixes)

		ip, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)

		_, err = ipam.DeletePrefix(ctx, prefix.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 192.168.0.0/20 has ips, delete prefix not possible", err.Error())

		// IPv6
		prefix, err = ipam.NewPrefix(ctx, "2001:0db8:85a3::/120")
		require.NoError(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, uint64(0), prefix.Usage().AcquiredPrefixes)

		ip, err = ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		require.NotNil(t, ip)

		_, err = ipam.DeletePrefix(ctx, prefix.Cidr)
		require.Error(t, err)
		require.Equal(t, "prefix 2001:db8:85a3::/120 has ips, delete prefix not possible", err.Error())

		_, err = ipam.ReleaseIP(ctx, ip)
		require.NoError(t, err)
		_, err = ipam.DeletePrefix(ctx, prefix.Cidr)
		require.NoError(t, err)
	})
}

func TestIpamer_PrefixFrom(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.PrefixFrom(ctx, "192.168.0.0/20")
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
		require.Nil(t, prefix)

		prefix, err = ipam.NewPrefix(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		prefix, err = ipam.PrefixFrom(ctx, "192.168.0.0/20")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		// non canonical form still returns the same prefix
		prefix2, err := ipam.PrefixFrom(ctx, "10.0.5.0/8")
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
		require.Nil(t, prefix2)

		prefix2a, err := ipam.NewPrefix(ctx, "10.8.0.0/8")
		require.NoError(t, err)
		require.NotNil(t, prefix2a)

		prefix2b, err := ipam.PrefixFrom(ctx, "10.2.0.0/8")
		require.NoError(t, err)
		require.NotNil(t, prefix2b)
		require.Equal(t, prefix2a.Cidr, prefix2b.Cidr)
	})
}

func TestIpamerAcquireIP(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "10.0.0.0/16"
		p, err := ipam.NewPrefix(ctx, cidr)
		require.NoError(t, err)
		for range 10 {
			if len(p.ips) != 2 {
				t.Fatalf("expected 2 ips in prefix, got %d", len(p.ips))
			}
			ip, err := ipam.AcquireIP(ctx, p.Cidr)
			require.NoError(t, err)
			require.NotNil(t, ip, "IP is nil")
			p, err = ipam.ReleaseIP(ctx, ip)
			require.NoError(t, err)
		}
		_, err = ipam.DeletePrefix(ctx, cidr)
		require.NoError(t, err, "error deleting prefix:%v", err)
	})
}

func TestIpamerAcquireIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "2001:0db8:85a3::/120"
		p, err := ipam.NewPrefix(ctx, cidr)
		require.NoError(t, err)
		for range 10 {
			if len(p.ips) != 1 {
				t.Fatalf("expected 1 ips in prefix, got %d", len(p.ips))
			}
			ip, err := ipam.AcquireIP(ctx, p.Cidr)
			require.NoError(t, err)
			require.NotNil(t, ip, "IP is nil")
			p, err = ipam.ReleaseIP(ctx, ip)
			require.NoError(t, err)
		}
		_, err = ipam.DeletePrefix(ctx, cidr)
		require.NoError(t, err, "error deleting prefix:%v", err)
	})
}
func TestIpamerAcquireAlreadyAcquiredIPv4(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "192.168.0.0/16"
		p, err := ipam.NewPrefix(ctx, cidr)
		require.NoError(t, err)
		ip, err := ipam.AcquireSpecificIP(ctx, p.Cidr, "192.168.2.4")
		require.NoError(t, err)
		require.NotNil(t, ip, "IP is nil")
		require.Equal(t, "192.168.2.4", ip.IP.String())
		_, err = ipam.AcquireSpecificIP(ctx, p.Cidr, "192.168.2.4")
		require.ErrorIs(t, err, ErrAlreadyAllocated)
		require.EqualError(t, err, "AlreadyAllocatedError: given ip:192.168.2.4 is already allocated")

		_, err = ipam.ReleaseIP(ctx, ip)
		require.NoError(t, err)
		_, err = ipam.DeletePrefix(ctx, cidr)
		require.NoError(t, err)
	})
}
func TestIpamerAcquireAlreadyAcquiredIPv6(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "2001:0db8:85a3::/64"
		p, err := ipam.NewPrefix(ctx, cidr)
		require.NoError(t, err)
		ip, err := ipam.AcquireSpecificIP(ctx, p.Cidr, "2001:0db8:85a3::1")
		require.NoError(t, err)
		require.NotNil(t, ip, "IP is nil")
		require.Equal(t, "2001:db8:85a3::1", ip.IP.String())
		_, err = ipam.AcquireSpecificIP(ctx, p.Cidr, "2001:0db8:85a3::1")
		require.ErrorIs(t, err, ErrAlreadyAllocated)
		require.EqualError(t, err, "AlreadyAllocatedError: given ip:2001:db8:85a3::1 is already allocated")

		_, err = ipam.ReleaseIP(ctx, ip)
		require.NoError(t, err)
		_, err = ipam.DeletePrefix(ctx, cidr)
		require.NoError(t, err)
	})
}
func TestGetHostAddresses(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "4.1.0.0/24"
		ips, err := ipam.getHostAddresses(ctx, cidr)
		require.NoError(t, err)
		require.NotNil(t, ips)
		require.Len(t, ips, 254)

		ip, err := ipam.AcquireIP(ctx, cidr)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoIPAvailable)
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 4.1.0.0/24 left, length of prefix.ips: 256", err.Error())
		require.Nil(t, ip)

		cidr = "3.1.0.0/26"
		ips, err = ipam.getHostAddresses(ctx, cidr)
		require.NoError(t, err)
		require.NotNil(t, ips)
		require.Len(t, ips, 62)

		ip, err = ipam.AcquireIP(ctx, cidr)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoIPAvailable)
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 3.1.0.0/26 left, length of prefix.ips: 64", err.Error())
		require.Nil(t, ip)
	})
}

func TestGetHostAddressesIPv6(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "2001:0db8:85a3::/120"
		ips, err := ipam.getHostAddresses(ctx, cidr)
		require.NoError(t, err)
		require.NotNil(t, ips)
		require.Len(t, ips, 255)

		ip, err := ipam.AcquireIP(ctx, cidr)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoIPAvailable)
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 2001:db8:85a3::/120 left, length of prefix.ips: 256", err.Error())
		require.Nil(t, ip)

		cidr = "2001:0db8:95a3::/122"
		ips, err = ipam.getHostAddresses(ctx, cidr)
		require.NoError(t, err)
		require.NotNil(t, ips)
		require.Len(t, ips, 63)

		ip, err = ipam.AcquireIP(ctx, cidr)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoIPAvailable)
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 2001:db8:95a3::/122 left, length of prefix.ips: 64", err.Error())
		require.Nil(t, ip)
	})
}
func TestPrefixDeepCopy(t *testing.T) {

	p1 := &Prefix{
		Cidr:                   "4.1.1.0/24",
		ParentCidr:             "4.1.0.0/16",
		availableChildPrefixes: map[string]bool{},
		isParent:               true,
		ips:                    map[string]bool{},
		version:                2,
	}

	p1.availableChildPrefixes["4.1.2.0/24"] = true
	p1.ips["4.1.1.1"] = true
	p1.ips["4.1.1.2"] = true

	p2 := p1.deepCopy()

	require.Equal(t, p1, p2)
	require.Equal(t, p1, p2)
	require.Equal(t, p1.availableChildPrefixes, p2.availableChildPrefixes)
	require.Equal(t, p1.ips, p2.ips)
}

func TestGob(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		data, err := prefix.GobEncode()
		require.NoError(t, err)

		newPrefix := &Prefix{}
		err = newPrefix.GobDecode(data)
		require.NoError(t, err)
		require.Equal(t, prefix, newPrefix)
	})
}

func TestPrefix_availablePrefixes(t *testing.T) {
	tests := []struct {
		name                   string
		cidr                   string
		availableChildPrefixes map[string]bool
		want                   uint64
	}{
		{
			name:                   "one child prefix",
			cidr:                   "192.168.0.0/20",
			availableChildPrefixes: map[string]bool{"192.168.0.0/22": false},
			want:                   512 + 256,
		},
		{
			name:                   "two child prefixes",
			cidr:                   "192.168.0.0/16",
			availableChildPrefixes: map[string]bool{"192.168.0.0/22": false, "192.168.0.0/26": false},
			want:                   8192 + 4096 + 2048 + 1024 + 512 + 256,
		},
		{
			name:                   "four child prefixes",
			cidr:                   "192.168.0.0/16",
			availableChildPrefixes: map[string]bool{"192.168.0.0/22": false, "192.168.0.0/26": false, "192.168.128.0/26": false, "192.168.196.0/26": false},
			want:                   4096 + 3*2048 + 3*1024 + 3*512 + 3*256 + 2*128 + 2*64 + 2*32 + 2*16,
		},
		{
			name: "simple ipv6",
			cidr: "2001:0db8:85a3::/120",
			want: 64,
		},
		{
			name:                   "one child prefix ipv6",
			cidr:                   "2001:0db8:85a3::/120",
			availableChildPrefixes: map[string]bool{"2001:0db8:85a3::/122": false},
			want:                   32 + 16,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr:                   test.cidr,
				availableChildPrefixes: test.availableChildPrefixes,
			}
			got, avpfxs := p.availablePrefixes()
			for _, pfx := range avpfxs {
				// Only logs if fails
				ipprefix, err := netip.ParsePrefix(pfx)
				require.NoError(t, err)
				smallest := 1 << (ipprefix.Addr().BitLen() - 2 - ipprefix.Bits())
				t.Logf("available prefix:%s smallest left:%d", pfx, smallest)
			}

			if test.want != got {
				t.Errorf("Prefix.availablePrefixes() = %d, want %d", got, test.want)
			}

			got2 := p.Usage().AvailableSmallestPrefixes
			if test.want != got2 {
				t.Errorf("Prefix.availablePrefixes() = %d, want %d", got2, test.want)
			}
		})
	}
}

func TestAcquireIPParallel(t *testing.T) {
	ctx := context.Background()
	ipsCount := 50
	g, _ := errgroup.WithContext(context.Background())

	mu := sync.Mutex{}
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		p, err := ipam.NewPrefix(ctx, "192.169.0.0/20")
		if err != nil {
			panic(err)
		}
		ips := make(map[string]bool)
		for range ipsCount {
			g.Go(func() error {
				ip, err := ipam.AcquireIP(ctx, p.Cidr)
				if err != nil {
					return err
				}
				if ip == nil {
					return fmt.Errorf("ip is nil")
				}

				mu.Lock()
				defer mu.Unlock()
				_, ok := ips[ip.IP.String()]
				if ok {
					return fmt.Errorf("duplicate ip:%s allocated", ip.IP.String())
				}
				ips[ip.IP.String()] = true

				return nil
			})
		}

		err = g.Wait()
		if err != nil {
			t.Fatal(err)
		}
		for ip := range ips {
			err := ipam.ReleaseIPFromPrefix(ctx, p.Cidr, ip)
			if err != nil {
				t.Error(err)
			}
		}
		_, err = ipam.DeletePrefix(ctx, p.Cidr)
		if err != nil {
			t.Error(err)
		}
	})
}

func Test_ipamer_DumpAndLoad(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
		require.NoError(t, err)
		require.NotNil(t, prefix)

		data, err := ipam.Dump(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		t.Log(data)

		err = ipam.Load(ctx, data)
		require.Error(t, err)
		require.Equal(t, "prefixes exist, please drop existing data before loading", err.Error())

		err = ipam.storage.DeleteAllPrefixes(ctx, defaultNamespace)
		require.NoError(t, err)
		err = ipam.Load(ctx, data)
		require.NoError(t, err)

		newPrefix, err := ipam.PrefixFrom(ctx, prefix.Cidr)
		require.NoError(t, err)

		require.Equal(t, prefix, newPrefix)
	})
}
func TestIpamer_ReadAllPrefixCidrs(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		const cidr = "192.168.0.0/20"

		prefix, err := ipam.NewPrefix(ctx, cidr)
		require.NoError(t, err)
		require.NotNil(t, prefix)

		cidrs, err := ipam.ReadAllPrefixCidrs(ctx)
		require.NoError(t, err)
		require.NotNil(t, cidrs)
		require.Len(t, cidrs, 1)
		require.Equal(t, cidr, cidrs[0])
	})
}

func TestIpamer_NamespacedOps(t *testing.T) {
	ctx := context.Background()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		err := ipam.CreateNamespace(ctx, "testns1")
		require.NoError(t, err)

		err = ipam.CreateNamespace(ctx, "testns2")
		require.NoError(t, err)

		ns, err := ipam.ListNamespaces(ctx)
		require.NoError(t, err)
		require.Len(t, ns, 3, "namespaces: %v", ns)
		require.Contains(t, ns, "root")
		require.Contains(t, ns, "testns1")
		require.Contains(t, ns, "testns2")

		// Create Prefix in a namespace
		createPrefixFn := func(ctx context.Context, namespace string) {
			ctx = NewContextWithNamespace(ctx, namespace)
			_, err := ipam.NewPrefix(ctx, "192.168.0.0/20")
			require.NoError(t, err)
			_, err = ipam.AcquireSpecificIP(ctx, "192.168.0.0/20", "192.168.0.2")
			require.NoError(t, err)
		}
		createPrefixFn(ctx, "testns1")
		createPrefixFn(ctx, "testns2")

		deletePrefixFn := func(ctx context.Context, namespace string) {
			ctx = NewContextWithNamespace(ctx, namespace)
			p, err := ipam.PrefixFrom(ctx, "192.168.0.0/20")
			require.NoError(t, err)
			require.NotNil(t, p)

			err = ipam.ReleaseIPFromPrefix(ctx, p.Cidr, "192.168.0.2")
			require.NoError(t, err)

			_, err = ipam.DeletePrefix(ctx, p.Cidr)
			require.NoError(t, err)
		}

		// Cannot delete namespace with allocated prefixes
		err = ipam.DeleteNamespace(ctx, "testns1")
		require.Error(t, err)

		// Delete prefixes first, then delete namespaces
		deletePrefixFn(ctx, "testns1")
		err = ipam.DeleteNamespace(ctx, "testns1")
		require.NoError(t, err)

		deletePrefixFn(ctx, "testns2")
		err = ipam.DeleteNamespace(ctx, "testns2")
		require.NoError(t, err)
	})
}

func TestPrefix_Network(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		want    netip.Addr
		wantErr bool
	}{
		{
			name:    "simple",
			cidr:    "192.168.0.0/16",
			want:    netip.MustParseAddr("192.168.0.0"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr: tt.cidr,
			}
			got, err := p.Network()
			if (err != nil) != tt.wantErr {
				t.Errorf("Prefix.Network() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Prefix.Network() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNamespaceFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "empty context",
			ctx:  context.Background(),
			want: defaultNamespace,
		},
		{
			name: "namespaced context",
			ctx:  NewContextWithNamespace(context.Background(), "a"),
			want: "a",
		},
		{
			name: "invalid context value",
			ctx:  context.WithValue(context.Background(), namespaceContextKey{}, true),
			want: defaultNamespace,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := namespaceFromContext(tt.ctx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("namespaceFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestAvailablePrefixes(t *testing.T) {
	testCases := []struct {
		name                 string
		cidr                 string
		expectedTotal        uint64
		expectedAvailablePfx []string
	}{
		{
			name:                 "192.168.0.0/32",
			cidr:                 "192.168.0.0/32",
			expectedTotal:        0,
			expectedAvailablePfx: []string{},
		},
		{
			name:                 "192.168.0.0/31",
			cidr:                 "192.168.0.0/31",
			expectedTotal:        0,
			expectedAvailablePfx: []string{},
		},
		{
			name:                 "192.168.0.0/30",
			cidr:                 "192.168.0.0/30",
			expectedTotal:        1,
			expectedAvailablePfx: []string{"192.168.0.0/30"},
		},
		{
			name:                 "192.168.0.0/24",
			cidr:                 "192.168.0.0/24",
			expectedTotal:        64,
			expectedAvailablePfx: []string{"192.168.0.0/24"},
		},
		{
			name:                 "2001:0db8:85a3::/128",
			cidr:                 "2001:0db8:85a3::/128",
			expectedTotal:        0,
			expectedAvailablePfx: []string{},
		},
		{
			name:                 "2001:0db8:85a3::/127",
			cidr:                 "2001:0db8:85a3::/127",
			expectedTotal:        0,
			expectedAvailablePfx: []string{},
		},
		{
			name:                 "2001:0db8:85a3::/126",
			cidr:                 "2001:0db8:85a3::/126",
			expectedTotal:        1,
			expectedAvailablePfx: []string{"2001:db8:85a3::/126"},
		},
		{
			name:                 "Invalid CIDR",
			cidr:                 "Invalid CIDR",
			expectedTotal:        0,
			expectedAvailablePfx: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prefix := &Prefix{
				Cidr:                   tc.cidr,
				isParent:               false,
				availableChildPrefixes: make(map[string]bool),
			}

			totalAvailable, availablePrefixes := prefix.availablePrefixes()

			assert.Equal(
				t, tc.expectedTotal, totalAvailable,
				"Expected totalAvailable: %d, got: %d",
				tc.expectedTotal, totalAvailable,
			)
			assert.ElementsMatchf(
				t, availablePrefixes, tc.expectedAvailablePfx,
				"Expected availablePrefixes: %v, got: %v",
				tc.expectedAvailablePfx, availablePrefixes,
			)
		})
	}
}

func TestChildPrefixParallel(t *testing.T) {
	ctx := context.Background()

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		parent, err := ipam.NewPrefix(ctx, "192.168.0.0/14")
		if err != nil {
			panic(err)
		}

		var (
			g1, _    = errgroup.WithContext(ctx)
			children []*Prefix
		)

		for i := range 100 {
			g1.Go(func() error {
				child, err := ipam.AcquireChildPrefix(ctx, parent.Cidr, 22)
				if err != nil {
					return fmt.Errorf("error acquiring prefix %d: %w", i, err)
				}

				children = append(children, child)

				return nil
			})
		}

		err = g1.Wait()
		require.NoError(t, err)

		g2, _ := errgroup.WithContext(ctx)

		for _, child := range children {
			g2.Go(func() error {
				err := ipam.ReleaseChildPrefix(ctx, child)
				if err != nil {
					return err
				}

				return nil
			})
		}

		err = g2.Wait()
		require.NoError(t, err)
	})
}
