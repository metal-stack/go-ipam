package ipam

import (
	"context"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

// start a tikv env with:
// tiup playground

func Test_tikv_CreatePrefix(t *testing.T) {
	var addrs []netip.AddrPort
	addrs = append(addrs, netip.MustParseAddrPort("127.0.0.1:2379"))
	tikv := newTikv(addrs)

	ctx := context.Background()
	pfxString := "192.168.0.0/24"

	pfx := Prefix{
		Cidr:                   pfxString,
		ParentCidr:             "",
		ips:                    make(map[string]bool),
		availableChildPrefixes: make(map[string]bool),
		isParent:               false,
	}
	_, err := tikv.DeletePrefix(ctx, pfx)
	require.NoError(t, err)

	got, err := tikv.CreatePrefix(ctx, pfx)
	require.NoError(t, err)

	read, err := tikv.ReadPrefix(ctx, pfxString)
	require.NoError(t, err)
	require.Equal(t, pfxString, read.Cidr)

	update := read
	update.ips = map[string]bool{"192.168.0.1": true}
	updated, err := tikv.UpdatePrefix(ctx, update)
	require.NoError(t, err)
	require.Equal(t, pfxString, updated.Cidr)
	require.Equal(t, map[string]bool{"192.168.0.1": true}, updated.ips)

	pfxs, err := tikv.ReadAllPrefixCidrs(ctx)
	require.NoError(t, err)
	require.Len(t, pfxs, 1)

	t.Log(got)
	t.Fail()
}

func BenchmarkTIKVNewPrefix(b *testing.B) {
	ctx := context.Background()
	var addrs []netip.AddrPort
	addrs = append(addrs, netip.MustParseAddrPort("127.0.0.1:2379"))
	tikv := newTikv(addrs)
	pfxString := "192.169.0.0/24"

	pfx := Prefix{
		Cidr:                   pfxString,
		ParentCidr:             "",
		ips:                    make(map[string]bool),
		availableChildPrefixes: make(map[string]bool),
		isParent:               false,
	}

	for n := 0; n < b.N; n++ {
		p, err := tikv.CreatePrefix(ctx, pfx)
		if err != nil {
			panic(err)
		}

		if p.Cidr == "" {
			panic("prefix is empty")
		}
		_, err = tikv.DeletePrefix(ctx, pfx)
		if err != nil {
			panic(err)
		}
	}
}
