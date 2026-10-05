package ipam

import (
	"context"
	"net/netip"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRangeOverlapping(t *testing.T) {
	ctx := t.Context()
	i := New(ctx)

	cidr := "10.10.10.0/24"
	_, err := i.NewPrefix(ctx, cidr)
	require.NoError(t, err)

	cidr = "10.10.10.1/24"
	_, err = i.NewPrefix(ctx, cidr)
	require.Error(t, err)
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
			if got := p.AvailableIPs(); got != test.want {
				t.Errorf("Prefix.Availableips() = %v, want %v", got, test.want)
			}
		})
	}
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

	p2 := p1.DeepCopy()

	require.Equal(t, p1, p2)
	require.Equal(t, p1, p2)
	require.Equal(t, p1.availableChildPrefixes, p2.availableChildPrefixes)
	require.Equal(t, p1.ips, p2.ips)
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
			got, avpfxs := p.AvailablePrefixes()
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
			want: DefaultNamespace,
		},
		{
			name: "namespaced context",
			ctx:  NewContextWithNamespace(t.Context(), "a"),
			want: "a",
		},
		{
			name: "invalid context value",
			ctx:  context.WithValue(t.Context(), namespaceContextKey{}, true),
			want: DefaultNamespace,
		},
	}
	for _, tt := range tests {
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

			totalAvailable, availablePrefixes := prefix.AvailablePrefixes()

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
