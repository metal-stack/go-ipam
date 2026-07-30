package ipam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPrefixNetworkAndBroadcastAllocatable(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.4.0/30", WithNetworkAndBroadcastAllocatable())
		require.NoError(t, err)
		require.True(t, prefix.NetworkAndBroadcastAllocatable())

		// network and broadcast are allocatable
		ip, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.4.0")
		require.NoError(t, err)
		require.Equal(t, "192.168.4.0", ip.IP.String())
		ip, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.4.3")
		require.NoError(t, err)
		require.Equal(t, "192.168.4.3", ip.IP.String())

		// all 4 addresses of the /30 are usable
		_, err = ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		_, err = ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		_, err = ipam.AcquireIP(ctx, prefix.Cidr)
		require.ErrorIs(t, err, ErrNoIPAvailable)
	})
}

func TestNewPrefixReservesNetworkAndBroadcastByDefault(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "192.168.5.0/30")
		require.NoError(t, err)
		require.False(t, prefix.NetworkAndBroadcastAllocatable())

		_, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.5.0")
		require.ErrorIs(t, err, ErrAlreadyAllocated)
		_, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "192.168.5.3")
		require.ErrorIs(t, err, ErrAlreadyAllocated)
	})
}

func TestSetPrefixNetworkAndBroadcastAllocatable(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "10.99.0.0/29")
		require.NoError(t, err)
		require.False(t, prefix.NetworkAndBroadcastAllocatable())

		// unreserving is always possible
		prefix, err = ipam.SetPrefixNetworkAndBroadcastAllocatable(ctx, prefix.Cidr, true)
		require.NoError(t, err)
		require.True(t, prefix.NetworkAndBroadcastAllocatable())

		ip, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "10.99.0.0")
		require.NoError(t, err)
		require.Equal(t, "10.99.0.0", ip.IP.String())
		ip, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "10.99.0.7")
		require.NoError(t, err)
		require.Equal(t, "10.99.0.7", ip.IP.String())

		// idempotent
		prefix, err = ipam.SetPrefixNetworkAndBroadcastAllocatable(ctx, prefix.Cidr, true)
		require.NoError(t, err)
		require.True(t, prefix.NetworkAndBroadcastAllocatable())
	})
}

func TestSetPrefixNetworkAndBroadcastNotAllocatable(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "10.98.0.0/29", WithNetworkAndBroadcastAllocatable())
		require.NoError(t, err)

		// network address given out, re-reserving must fail
		_, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "10.98.0.0")
		require.NoError(t, err)
		_, err = ipam.SetPrefixNetworkAndBroadcastAllocatable(ctx, prefix.Cidr, false)
		require.ErrorIs(t, err, ErrAlreadyAllocated)

		// after releasing it, re-reserving works
		err = ipam.ReleaseIPFromPrefix(ctx, prefix.Cidr, "10.98.0.0")
		require.NoError(t, err)
		prefix, err = ipam.SetPrefixNetworkAndBroadcastAllocatable(ctx, prefix.Cidr, false)
		require.NoError(t, err)
		require.False(t, prefix.NetworkAndBroadcastAllocatable())

		_, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "10.98.0.0")
		require.ErrorIs(t, err, ErrAlreadyAllocated)
		_, err = ipam.AcquireSpecificIP(ctx, prefix.Cidr, "10.98.0.7")
		require.ErrorIs(t, err, ErrAlreadyAllocated)

		// idempotent
		prefix, err = ipam.SetPrefixNetworkAndBroadcastAllocatable(ctx, prefix.Cidr, false)
		require.NoError(t, err)
		require.False(t, prefix.NetworkAndBroadcastAllocatable())
	})
}

func TestAllocatablePrefixDeleteGuard(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "10.97.0.0/30", WithNetworkAndBroadcastAllocatable())
		require.NoError(t, err)

		// a single allocated ip must prevent deletion
		ip, err := ipam.AcquireIP(ctx, prefix.Cidr)
		require.NoError(t, err)
		_, err = ipam.DeletePrefix(ctx, prefix.Cidr)
		require.Error(t, err)

		_, err = ipam.ReleaseIP(ctx, ip)
		require.NoError(t, err)
		_, err = ipam.DeletePrefix(ctx, prefix.Cidr)
		require.NoError(t, err)
	})
}

func TestNetworkAndBroadcastAllocatableIPv6(t *testing.T) {
	ctx := t.Context()
	testWithBackends(t, func(t *testing.T, ipam *ipamer) {
		prefix, err := ipam.NewPrefix(ctx, "2001:db8:99::/126", WithNetworkAndBroadcastAllocatable())
		require.NoError(t, err)

		// the network address is allocatable, all 4 addresses are usable
		ip, err := ipam.AcquireSpecificIP(ctx, prefix.Cidr, "2001:db8:99::")
		require.NoError(t, err)
		require.Equal(t, "2001:db8:99::", ip.IP.String())
		for range 3 {
			_, err = ipam.AcquireIP(ctx, prefix.Cidr)
			require.NoError(t, err)
		}
		_, err = ipam.AcquireIP(ctx, prefix.Cidr)
		require.ErrorIs(t, err, ErrNoIPAvailable)
	})
}

func TestNetworkAndBroadcastAllocatableJSONRoundTrip(t *testing.T) {
	p := &Prefix{
		Cidr:                           "192.168.6.0/24",
		ips:                            map[string]bool{},
		availableChildPrefixes:         map[string]bool{},
		networkAndBroadcastAllocatable: true,
	}
	js, err := p.toJSON()
	require.NoError(t, err)
	restored, err := fromJSON(js)
	require.NoError(t, err)
	require.True(t, restored.NetworkAndBroadcastAllocatable())

	// default stays reserved
	p.networkAndBroadcastAllocatable = false
	js, err = p.toJSON()
	require.NoError(t, err)
	restored, err = fromJSON(js)
	require.NoError(t, err)
	require.False(t, restored.NetworkAndBroadcastAllocatable())
}
