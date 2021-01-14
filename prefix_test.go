package ipam

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"inet.af/netaddr"
)

func mustIP(s string) netaddr.IP {
	ip, err := netaddr.ParseIP(s)
	if err != nil {
		panic(err)
	}

	return ip
}

// getHostAddresses will return all possible ipadresses a host can get in the given prefix.
// The IPs will be acquired by this method, so that the prefix has no free IPs afterwards.
func (i *ipamer) getHostAddresses(prefix string) ([]string, error) {
	hostAddresses := []string{}

	p, err := i.NewPrefix(prefix)
	if err != nil {
		return hostAddresses, err
	}

	// loop till AcquireIP signals that it has no ips left
	for {
		ip, err := i.AcquireIP(p.Cidr)
		if errors.Is(err, ErrNoIPAvailable) {
			return hostAddresses, nil
		}
		if err != nil {
			return nil, err
		}
		hostAddresses = append(hostAddresses, ip.IP.String())
	}
}

func TestIpamer_AcquireIP(t *testing.T) {

	type fields struct {
		prefixCIDR  string
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
			want: &IP{IP: mustIP("192.168.1.1"), ParentPrefix: "192.168.1.0/24"},
		},
		{
			name: "Acquire next IPv6 regularly",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/124",
				existingips: []string{},
			},
			want: &IP{IP: mustIP("2001:0db8:85a3::1"), ParentPrefix: "2001:0db8:85a3::/124"},
		},
		{
			name: "Want next IP, network already occupied a little",
			fields: fields{
				prefixCIDR:  "192.168.2.0/30",
				existingips: []string{"192.168.2.1"},
			},
			want: &IP{IP: mustIP("192.168.2.2"), ParentPrefix: "192.168.2.0/30"},
		},
		{
			name: "Want next IPv6, network already occupied a little",
			fields: fields{
				prefixCIDR:  "2001:0db8:85a3::/124",
				existingips: []string{"2001:db8:85a3::1"},
			},
			want: &IP{IP: mustIP("2001:db8:85a3::2"), ParentPrefix: "2001:0db8:85a3::/124"},
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

		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			p, err := ipam.NewPrefix(tt.fields.prefixCIDR)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			for _, ipString := range tt.fields.existingips {
				p.ips[ipString] = true
			}

			var updatedPrefix Prefix
			updatedPrefix, err = ipam.storage.UpdatePrefix(*p)
			if err != nil {
				t.Errorf("Could not update prefix: %v", err)
			}
			got, _ := ipam.AcquireIP(updatedPrefix.Cidr)
			if tt.want == nil || got == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Ipamer.AcquireIP() want or got is nil, got %v, want %v", got, tt.want)
				}
			} else {
				if tt.want.IP.Compare(got.IP) != 0 {
					t.Errorf("Ipamer.AcquireIP() got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIpamer_ReleaseIPFromPrefixIPv4(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		err = ipam.ReleaseIPFromPrefix(prefix.Cidr, "1.2.3.4")
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to release ip:1.2.3.4 because it is not allocated in prefix:192.168.0.0/24", err.Error())

		err = ipam.ReleaseIPFromPrefix("4.5.6.7/23", "1.2.3.4")
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to find prefix for cidr:4.5.6.7/23", err.Error())
	})
}

func TestIpamer_ReleaseIPFromPrefixIPv6(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("2001:0db8:85a3::/120")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		err = ipam.ReleaseIPFromPrefix(prefix.Cidr, "1.2.3.4")
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to release ip:1.2.3.4 because it is not allocated in prefix:2001:0db8:85a3::/120", err.Error())

		err = ipam.ReleaseIPFromPrefix(prefix.Cidr, "1001:0db8:85a3::1")
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to release ip:1001:0db8:85a3::1 because it is not allocated in prefix:2001:0db8:85a3::/120", err.Error())

		err = ipam.ReleaseIPFromPrefix("1001:0db8:85a3::/120", "1.2.3.4")
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to find prefix for cidr:1001:0db8:85a3::/120", err.Error())
	})
}
func TestIpamer_AcquireSpecificIP(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		// IPv4
		prefix, err := ipam.NewPrefix("192.168.99.0/24")
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		// network an broadcast are blocked
		require.Equal(t, prefix.acquiredips(), uint64(2))
		ip1, err := ipam.AcquireSpecificIP(prefix.Cidr, "192.168.99.1")
		require.Nil(t, err)
		require.NotNil(t, ip1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))
		ip2, err := ipam.AcquireSpecificIP(prefix.Cidr, "192.168.99.2")
		require.Nil(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(4))
		require.Equal(t, "192.168.99.1", ip1.IP.String())
		require.Equal(t, "192.168.99.2", ip2.IP.String())

		// Wish IP out of prefix
		ip3, err := ipam.AcquireSpecificIP(prefix.Cidr, "192.168.98.2")
		require.Nil(t, ip3)
		require.NotNil(t, err)
		require.Equal(t, "given ip:192.168.98.2 is not in 192.168.99.0/24", err.Error())

		// Wish IP is invalid
		ip4, err := ipam.AcquireSpecificIP(prefix.Cidr, "192.168.99.1.invalid")
		require.Nil(t, ip4)
		require.NotNil(t, err)
		require.Equal(t, "given ip:192.168.99.1.invalid in not valid", err.Error())

		// Cidr is invalid
		ip5, err := ipam.AcquireSpecificIP("3.4.5.6/27", "192.168.99.1.invalid")
		require.Nil(t, ip5)
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to find prefix for cidr:3.4.5.6/27", err.Error())

		prefix, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))

		prefix, err = ipam.ReleaseIP(ip2)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))

		// IPv6
		prefix, err = ipam.NewPrefix("2001:0db8:85a3::/120")
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		// network is blocked
		require.Equal(t, prefix.acquiredips(), uint64(1))
		ip1, err = ipam.AcquireSpecificIP(prefix.Cidr, "2001:db8:85a3::1")
		require.Nil(t, err)
		require.NotNil(t, ip1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))
		ip2, err = ipam.AcquireSpecificIP(prefix.Cidr, "2001:0db8:85a3::2")
		require.Nil(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))
		require.Equal(t, "2001:db8:85a3::1", ip1.IP.String())
		require.Equal(t, "2001:db8:85a3::2", ip2.IP.String())

		// Wish IP out of prefix
		ip3, err = ipam.AcquireSpecificIP(prefix.Cidr, "2001:0db8:85a4::1")
		require.Nil(t, ip3)
		require.NotNil(t, err)
		require.Equal(t, "given ip:2001:0db8:85a4::1 is not in 2001:0db8:85a3::/120", err.Error())

		// Cidr is invalid
		ip5, err = ipam.AcquireSpecificIP("2001:0db8:95a3::/120", "2001:0db8:95a3::invalid")
		require.Nil(t, ip5)
		require.NotNil(t, err)
		require.True(t, errors.As(err, &NotFoundError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNotFound), "error must be NotFound")
		require.Equal(t, "NotFound: unable to find prefix for cidr:2001:0db8:95a3::/120", err.Error())

		prefix, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))

		prefix, err = ipam.ReleaseIP(ip2)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(1))
	})
}

func TestIpamer_AcquireIPCountsIPv4(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		// network an broadcast are blocked
		require.Equal(t, prefix.acquiredips(), uint64(2))
		ip1, err := ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))
		ip2, err := ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(4))
		require.True(t, strings.HasPrefix(ip1.IP.String(), "192.168.0"))
		require.True(t, strings.HasPrefix(ip2.IP.String(), "192.168.0"))

		prefix, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))

		prefix, err = ipam.ReleaseIP(ip2)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))
	})
}

func TestIpamer_AcquireIPCountsIPv6(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("2001:0db8:85a3::/120")
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		// network is blocked
		require.Equal(t, prefix.acquiredips(), uint64(1))
		ip1, err := ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))
		ip2, err := ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotEqual(t, ip1, ip2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))
		require.True(t, strings.HasPrefix(ip1.IP.String(), "2001:db8:85a3::"))
		require.True(t, strings.HasPrefix(ip2.IP.String(), "2001:db8:85a3::"))

		prefix, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))

		prefix, err = ipam.ReleaseIP(ip2)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(1))
	})
}

func TestIpamer_AcquireChildPrefixFragmented(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		allPrefixes, err := ipam.storage.ReadAllPrefixes()
		require.NoError(t, err)
		require.Equal(t, 0, len(allPrefixes))

		// Create Prefix with /20
		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.NoError(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, 1024, int(s))
		require.Equal(t, 0, int(prefix.acquiredPrefixes()))
		require.Equal(t, 0, int(prefix.Usage().AcquiredPrefixes))

		// Acquire first half 192.168.0.0/21 = 192.168.0.0 - 192.168.7.254
		c1, err := ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.NoError(t, err)
		require.NotNil(t, c1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, 512, int(s))
		require.Equal(t, 1, int(prefix.acquiredPrefixes()))
		require.Equal(t, 1, int(prefix.Usage().AcquiredPrefixes))

		// acquire 1/4the of the rest 192.168.8.0/22 = 192.168.8.0 - 192.168.11.254
		c2, err := ipam.AcquireChildPrefix(prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		s, a := prefix.availablePrefixes()
		// Next free must be 192.168.12.0/22
		require.Equal(t, []string{"192.168.12.0/22"}, a)
		require.Equal(t, 256, int(s))
		require.Equal(t, 2, int(prefix.acquiredPrefixes()))
		require.Equal(t, 2, int(prefix.Usage().AcquiredPrefixes))

		// acquire impossible size
		_, err = ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.EqualError(t, err, "no prefix found in 192.168.0.0/20 with length:21, but 192.168.12.0/22 is available")

		// Release small, first half acquired
		err = ipam.ReleaseChildPrefix(c2)
		require.NoError(t, err)

		// acquire /28
		c3, err := ipam.AcquireChildPrefix(prefix.Cidr, 28)
		require.NoError(t, err)
		require.NotNil(t, c3)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		s, a = prefix.availablePrefixes()
		require.Equal(t, []string{"192.168.8.16/28", "192.168.8.32/27", "192.168.8.64/26", "192.168.8.128/25", "192.168.9.0/24", "192.168.10.0/23", "192.168.12.0/22"}, a)
		require.Equal(t, 508, int(s))
		require.Equal(t, 2, int(prefix.acquiredPrefixes()))
		require.Equal(t, 2, int(prefix.Usage().AcquiredPrefixes))

		// acquire impossible size
		_, err = ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.EqualError(t, err, "no prefix found in 192.168.0.0/20 with length:21, but 192.168.8.16/28,192.168.8.32/27,192.168.8.64/26,192.168.8.128/25,192.168.9.0/24,192.168.10.0/23,192.168.12.0/22 are available")

		// acquire a /22 which must be possible
		c4, err := ipam.AcquireChildPrefix(prefix.Cidr, 22)
		require.NoError(t, err)
		require.NotNil(t, c4)
		require.Equal(t, c4.String(), "192.168.12.0/22")

	})
}

func TestIpamer_AcquireChildPrefixCounts(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		allPrefixes, err := ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 0, len(allPrefixes))

		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, s, uint64(1024))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))

		usage := prefix.Usage()
		require.Equal(t, "ip:2/4096", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))

		c1, err := ipam.AcquireChildPrefix(prefix.Cidr, 22)
		require.Nil(t, err)
		require.NotNil(t, c1)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(768), s)
		require.Equal(t, uint64(1), prefix.acquiredPrefixes())
		require.Equal(t, uint64(1), prefix.Usage().AcquiredPrefixes)

		usage = prefix.Usage()
		require.Equal(t, "ip:2/4096 prefixes alloc:1 avail:768", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 2, len(allPrefixes))

		c2, err := ipam.AcquireChildPrefix(prefix.Cidr, 22)
		require.Nil(t, err)
		require.NotNil(t, c2)
		prefix = ipam.PrefixFrom(prefix.Cidr)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(512), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(2))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(2))
		require.True(t, strings.HasSuffix(c1.Cidr, "/22"))
		require.True(t, strings.HasSuffix(c2.Cidr, "/22"))
		require.True(t, strings.HasPrefix(c1.Cidr, "192.168."))
		require.True(t, strings.HasPrefix(c2.Cidr, "192.168."))
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 3, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c1)
		prefix = ipam.PrefixFrom(prefix.Cidr)

		require.Nil(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(768), s)
		require.Equal(t, uint64(1), prefix.acquiredPrefixes())
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 2, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c2)
		prefix = ipam.PrefixFrom(prefix.Cidr)

		require.Nil(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c1)
		require.Errorf(t, err, "unable to release prefix %s:delete prefix:%s not found", c1.Cidr)

		c3, err := ipam.AcquireChildPrefix(prefix.Cidr, 22)
		require.Nil(t, err)
		require.NotNil(t, c2)
		ip1, err := ipam.AcquireIP(c3.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip1)

		err = ipam.ReleaseChildPrefix(c3)
		require.Errorf(t, err, "prefix %s has ips, deletion not possible", c3.Cidr)

		c3, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)

		err = ipam.ReleaseChildPrefix(c3)
		require.Nil(t, err)

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))
	})
}

func TestIpamer_AcquireChildPrefixIPv4(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))

		// Same length
		cp, err := ipam.AcquireChildPrefix(prefix.Cidr, 20)
		require.NotNil(t, err)
		require.Equal(t, "given length:20 must be greater than prefix length:20", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.Nil(t, err)
		require.NotNil(t, cp)
		require.True(t, strings.HasPrefix(cp.Cidr, "192.168."))
		require.True(t, strings.HasSuffix(cp.Cidr, "/21"))
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// No more ChildPrefixes
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.Nil(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 21)
		require.NotNil(t, err)
		require.Equal(t, "no prefix found in 192.168.0.0/20 with length:21", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix("10.0.0.0/24")
		require.Nil(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, p2.acquiredPrefixes(), uint64(0))
		ip, err := ipam.AcquireIP(p2.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireChildPrefix(p2.Cidr, 25)
		require.NotNil(t, err)
		require.Equal(t, "prefix 10.0.0.0/24 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Childs, AcquireIP wont work
		p3, err := ipam.NewPrefix("172.17.0.0/24")
		require.Nil(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, p3.acquiredPrefixes(), uint64(0))
		cp3, err := ipam.AcquireChildPrefix(p3.Cidr, 25)
		require.Nil(t, err)
		require.NotNil(t, cp3)
		p3 = ipam.PrefixFrom(p3.Cidr)
		ip, err = ipam.AcquireIP(p3.Cidr)
		require.NotNil(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(p3)
		require.NotNil(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 is no child prefix", err.Error())
	})
}

func TestIpamer_AcquireChildPrefixIPv6(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("2001:0db8:85a3::/116")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))

		// Same length
		cp, err := ipam.AcquireChildPrefix(prefix.Cidr, 116)
		require.NotNil(t, err)
		require.Equal(t, "given length:116 must be greater than prefix length:116", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 117)
		require.Nil(t, err)
		require.NotNil(t, cp)
		require.True(t, strings.HasPrefix(cp.Cidr, "2001:db8:85a3:"))
		require.True(t, strings.HasSuffix(cp.Cidr, "/117"))
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// No more ChildPrefixes
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 117)
		require.Nil(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireChildPrefix(prefix.Cidr, 117)
		require.NotNil(t, err)
		require.Equal(t, "no prefix found in 2001:0db8:85a3::/116 with length:117", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix("2001:0db8:95a3::/120")
		require.Nil(t, err)
		s, _ = p2.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, p2.acquiredPrefixes(), uint64(0))
		ip, err := ipam.AcquireIP(p2.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireChildPrefix(p2.Cidr, 121)
		require.NotNil(t, err)
		require.Equal(t, "prefix 2001:0db8:95a3::/120 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Childs, AcquireIP wont work
		p3, err := ipam.NewPrefix("2001:0db8:75a3::/120")
		require.Nil(t, err)
		s, _ = p3.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, p3.acquiredPrefixes(), uint64(0))
		cp3, err := ipam.AcquireChildPrefix(p3.Cidr, 121)
		require.Nil(t, err)
		require.NotNil(t, cp3)
		p3 = ipam.PrefixFrom(p3.Cidr)
		ip, err = ipam.AcquireIP(p3.Cidr)
		require.NotNil(t, err)
		require.Equal(t, "prefix 2001:0db8:75a3::/120 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(p3)
		require.NotNil(t, err)
		require.Equal(t, "prefix 2001:0db8:75a3::/120 is no child prefix", err.Error())
	})
}

func TestIpamer_AcquireChildPrefixNoDuplicatesUntilFullIPv6(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("2001:0db8:85a3::/112")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(16384), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		uniquePrefixes := make(map[string]bool)
		// acquire all /120 prefixes (2^8 = 256)
		for i := 0; i < 256; i++ {
			cp, err := ipam.AcquireChildPrefix(prefix.Cidr, 120)
			require.Nil(t, err)
			require.NotNil(t, cp)
			require.True(t, strings.HasPrefix(cp.Cidr, "2001:db8:85a3:"))
			require.True(t, strings.HasSuffix(cp.Cidr, "/120"))
			require.Equal(t, prefix.Cidr, cp.ParentCidr)
			_, ok := uniquePrefixes[cp.String()]
			require.False(t, ok)
			uniquePrefixes[cp.String()] = true
		}
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, 256, len(uniquePrefixes))
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(0), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(256))

	})
}
func TestIpamer_AcquireChildPrefixNoDuplicatesUntilFullIPv4(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/16")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(16384), s)
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())

		uniquePrefixes := make(map[string]bool)
		// acquire all /24 prefixes (2^8 = 256)
		for i := 0; i < 256; i++ {
			cp, err := ipam.AcquireChildPrefix(prefix.Cidr, 24)
			require.Nil(t, err)
			require.NotNil(t, cp)
			require.True(t, strings.HasPrefix(cp.Cidr, "192.168."))
			require.True(t, strings.HasSuffix(cp.Cidr, "/24"))
			require.Equal(t, prefix.Cidr, cp.ParentCidr)
			_, ok := uniquePrefixes[cp.String()]
			require.False(t, ok)
			uniquePrefixes[cp.String()] = true
		}
		prefix = ipam.PrefixFrom(prefix.Cidr)
		require.Equal(t, 256, len(uniquePrefixes))
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(0), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(256))

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
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr: tt.Cidr,
			}
			if got := p.availableips(); got != tt.want {
				t.Errorf("Prefix.Availableips() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpamer_PrefixesOverlapping(t *testing.T) {

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
			errorString:      "2001:0db8:85a4::/126 overlaps 2001:0db8:85a4::/126",
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
		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			for _, ep := range tt.existingPrefixes {
				p, err := ipam.NewPrefix(ep)
				if err != nil {
					t.Errorf("Newprefix on ExistingPrefix failed:%v", err)
				}
				if p == nil {
					t.Errorf("Newprefix on ExistingPrefix returns nil")
				}
			}
			err := ipam.PrefixesOverlapping(tt.existingPrefixes, tt.newPrefixes)
			if tt.wantErr && err == nil {
				t.Errorf("Ipamer.PrefixesOverlapping() expected error but err was nil")
			}
			if tt.wantErr && err != nil && err.Error() != tt.errorString {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v, errorString = %v", err, tt.wantErr, tt.errorString)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIpamer_NewPrefix(t *testing.T) {

	tests := []struct {
		name        string
		cidr        string
		wantErr     bool
		errorString string
	}{
		{
			name:    "valid Prefix",
			cidr:    "192.168.0.0/24",
			wantErr: false,
		},
		{
			name:        "invalid Prefix",
			cidr:        "192.168.0.0/33",
			wantErr:     true,
			errorString: "unable to parse cidr:192.168.0.0/33 netaddr.ParseIPPrefix(\"33\"): prefix length out of range",
		},
		{
			name:    "valid IPv6 Prefix",
			cidr:    "2001:0db8:85a3::/120",
			wantErr: false,
		},
		{
			name:        "invalid IPv6 Prefix length",
			cidr:        "2001:0db8:85a3::/129",
			wantErr:     true,
			errorString: "unable to parse cidr:2001:0db8:85a3::/129 netaddr.ParseIPPrefix(\"129\"): prefix length out of range",
		},
		{
			name:        "invalid IPv6 Prefix length",
			cidr:        "2001:0db8:85a3:::/120",
			wantErr:     true,
			errorString: "unable to parse cidr:2001:0db8:85a3:::/120 netaddr.ParseIPPrefix(\"2001:0db8:85a3:::/120\"): ParseIP(\"2001:0db8:85a3:::\"): each colon-separated field must have at least one digit (at \":\")",
		},
	}
	for _, tt := range tests {

		testWithBackends(t, func(t *testing.T, ipam *ipamer) {
			got, err := ipam.NewPrefix(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ipamer.NewPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (err != nil) && tt.errorString != err.Error() {
				t.Errorf("Ipamer.NewPrefix() error = %v, errorString %v", err, tt.errorString)
				return
			}

			if err != nil {
				return
			}
			if !reflect.DeepEqual(got.Cidr, tt.cidr) {
				t.Errorf("Ipamer.NewPrefix() = %v, want %v", got.Cidr, tt.cidr)
			}
		})
	}
}

func TestIpamer_DeletePrefix(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		// IPv4
		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		s, _ := prefix.availablePrefixes()
		require.Equal(t, uint64(1024), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))

		ip, err := ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip)

		_, err = ipam.DeletePrefix(prefix.Cidr)
		require.NotNil(t, err)
		require.Equal(t, "prefix 192.168.0.0/20 has ips, delete prefix not possible", err.Error())

		// IPv6
		prefix, err = ipam.NewPrefix("2001:0db8:85a3::/120")
		require.Nil(t, err)
		s, _ = prefix.availablePrefixes()
		require.Equal(t, uint64(64), s)
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))

		ip, err = ipam.AcquireIP(prefix.Cidr)
		require.Nil(t, err)
		require.NotNil(t, ip)

		_, err = ipam.DeletePrefix(prefix.Cidr)
		require.NotNil(t, err)
		require.Equal(t, "prefix 2001:0db8:85a3::/120 has ips, delete prefix not possible", err.Error())

		_, err = ipam.ReleaseIP(ip)
		require.Nil(t, err)
		_, err = ipam.DeletePrefix(prefix.Cidr)
		require.Nil(t, err)
	})
}

func TestIpamer_PrefixFrom(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix := ipam.PrefixFrom("192.168.0.0/20")
		require.Nil(t, prefix)

		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		prefix = ipam.PrefixFrom("192.168.0.0/20")
		require.NotNil(t, prefix)
	})
}

func TestIpamerAcquireIP(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "10.0.0.0/16"
		p, err := ipam.NewPrefix(cidr)
		if err != nil {
			panic(err)
		}
		for n := 0; n < 10; n++ {
			if len(p.ips) != 2 {
				t.Fatalf("expected 2 ips in prefix, got %d", len(p.ips))
			}
			ip, err := ipam.AcquireIP(p.Cidr)
			if err != nil {
				panic(err)
			}
			if ip == nil {
				panic("IP nil")
			}
			p, err = ipam.ReleaseIP(ip)
			if err != nil {
				panic(err)
			}
		}
		_, err = ipam.DeletePrefix(cidr)
		if err != nil {
			t.Errorf("error deleting prefix:%v", err)
		}
	})
}

func TestIpamerAcquireIPv6(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "2001:0db8:85a3::/120"
		p, err := ipam.NewPrefix(cidr)
		if err != nil {
			panic(err)
		}
		for n := 0; n < 10; n++ {
			if len(p.ips) != 1 {
				t.Fatalf("expected 1 ips in prefix, got %d", len(p.ips))
			}
			ip, err := ipam.AcquireIP(p.Cidr)
			if err != nil {
				panic(err)
			}
			if ip == nil {
				panic("IP nil")
			}
			p, err = ipam.ReleaseIP(ip)
			if err != nil {
				panic(err)
			}
		}
		_, err = ipam.DeletePrefix(cidr)
		if err != nil {
			t.Errorf("error deleting prefix:%v", err)
		}
	})
}

func TestGetHostAddresses(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "4.1.0.0/24"
		ips, err := ipam.getHostAddresses(cidr)
		if err != nil {
			panic(err)
		}
		require.NotNil(t, ips)
		require.Equal(t, 254, len(ips))

		ip, err := ipam.AcquireIP(cidr)
		require.Error(t, err)
		require.True(t, errors.As(err, &NoIPAvailableError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNoIPAvailable), "error must be NoIPAvailable")
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 4.1.0.0/24 left, length of prefix.ips: 256", err.Error())
		require.Nil(t, ip)

		cidr = "3.1.0.0/26"
		ips, err = ipam.getHostAddresses(cidr)
		if err != nil {
			panic(err)
		}
		require.NotNil(t, ips)
		require.Equal(t, 62, len(ips))

		ip, err = ipam.AcquireIP(cidr)
		require.Error(t, err)
		require.True(t, errors.As(err, &NoIPAvailableError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNoIPAvailable), "error must be NoIPAvailable")
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 3.1.0.0/26 left, length of prefix.ips: 64", err.Error())
		require.Nil(t, ip)
	})
}

func TestGetHostAddressesIPv6(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		cidr := "2001:0db8:85a3::/120"
		ips, err := ipam.getHostAddresses(cidr)
		if err != nil {
			panic(err)
		}
		require.NotNil(t, ips)
		require.Equal(t, 255, len(ips))

		ip, err := ipam.AcquireIP(cidr)
		require.Error(t, err)
		require.True(t, errors.As(err, &NoIPAvailableError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNoIPAvailable), "error must be NoIPAvailable")
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 2001:0db8:85a3::/120 left, length of prefix.ips: 256", err.Error())
		require.Nil(t, ip)

		cidr = "2001:0db8:95a3::/122"
		ips, err = ipam.getHostAddresses(cidr)
		if err != nil {
			panic(err)
		}
		require.NotNil(t, ips)
		require.Equal(t, 63, len(ips))

		ip, err = ipam.AcquireIP(cidr)
		require.Error(t, err)
		require.True(t, errors.As(err, &NoIPAvailableError{}), "error must be of correct type")
		require.True(t, errors.Is(err, ErrNoIPAvailable), "error must be NoIPAvailable")
		require.Equal(t, "NoIPAvailableError: no more ips in prefix: 2001:0db8:95a3::/122 left, length of prefix.ips: 64", err.Error())
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

	require.False(t, p1 == p2)
	require.Equal(t, p1, p2)
	require.False(t, &(p1.availableChildPrefixes) == &(p2.availableChildPrefixes))
	require.False(t, &(p1.ips) == &(p2.ips))
}

func TestGob(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		data, err := prefix.GobEncode()
		require.Nil(t, err)

		newPrefix := &Prefix{}
		err = newPrefix.GobDecode(data)
		require.Nil(t, err)
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
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr:                   tt.cidr,
				availableChildPrefixes: tt.availableChildPrefixes,
			}
			got, avpfxs := p.availablePrefixes()
			for _, pfx := range avpfxs {
				// Only logs if fails
				ipprefix, err := netaddr.ParseIPPrefix(pfx)
				require.NoError(t, err)
				smallest := 1 << (ipprefix.IP.BitLen() - 2 - ipprefix.Bits)
				t.Logf("available prefix:%s smallest left:%d", pfx, smallest)
			}

			if tt.want != got {
				t.Errorf("Prefix.availablePrefixes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		data, err := prefix.JSONEncode()
		require.Nil(t, err)

		newPrefix := &Prefix{}
		err = newPrefix.JSONDecode(data)
		require.Nil(t, err)
		require.Equal(t, prefix, newPrefix)
	})
}
