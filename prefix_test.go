package ipam

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"strings"

	"github.com/stretchr/testify/require"
)

func TestIpamer_AcquireIP(t *testing.T) {
	type fields struct {
		storage     Storage
		prefixCIDR  string
		existingIPs []string
	}
	tests := []struct {
		name   string
		fields fields
		want   *IP
	}{
		{
			name: "Acquire next IP regularly",
			fields: fields{
				storage:     memory{},
				prefixCIDR:  "192.168.1.0/24",
				existingIPs: []string{},
			},
			want: &IP{IP: net.ParseIP("192.168.1.1")},
		},
		{
			name: "Want next IP, network already occupied a little",
			fields: fields{
				storage:     memory{},
				prefixCIDR:  "192.168.2.0/30",
				existingIPs: []string{"192.168.2.1"},
			},
			want: &IP{IP: net.ParseIP("192.168.2.2")},
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				storage:     memory{},
				prefixCIDR:  "192.168.3.0/30",
				existingIPs: []string{"192.168.3.1", "192.168.3.2"},
			},
			want: nil,
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				storage:    memory{},
				prefixCIDR: "192.168.4.0/32",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Ipamer{
				storage: tt.fields.storage,
			}
			p, err := i.NewPrefix(tt.fields.prefixCIDR)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			for _, ipString := range tt.fields.existingIPs {
				fmt.Printf("existing:%s\n", ipString)
				i := net.ParseIP(ipString)
				p.ips[ipString] = IP{IP: i}
			}
			got, _ := i.AcquireIP(*p)
			if tt.want == nil || got == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Ipamer.AcquireIP() = %v, want %v", got, tt.want)
				}
			} else {
				if !tt.want.IP.Equal(got.IP) {
					t.Errorf("Ipamer.AcquireIP() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIpamer_AcquireIPMulti(t *testing.T) {
	ipam := New()

	prefix, err := ipam.NewPrefix("192.168.0.0/24")
	require.Nil(t, err)
	require.Equal(t, prefix.AvailableIPs(), int64(256))
	// network an broadcast are blocked
	require.Equal(t, prefix.AcquiredIPs(), 2)
	ip1, err := ipam.AcquireIP(*prefix)
	require.Nil(t, err)
	require.NotNil(t, ip1)
	require.Equal(t, prefix.AvailableIPs(), int64(256))
	require.Equal(t, prefix.AcquiredIPs(), 3)
	ip2, err := ipam.AcquireIP(*prefix)
	require.NotEqual(t, ip1, ip2)
	require.Equal(t, prefix.AvailableIPs(), int64(256))
	require.Equal(t, prefix.AcquiredIPs(), 4)
	require.True(t, strings.HasPrefix(ip1.IP.String(), "192.168.0"))
	require.True(t, strings.HasPrefix(ip2.IP.String(), "192.168.0"))

	err = ipam.ReleaseIP(*ip1)
	require.Nil(t, err)
	require.Equal(t, prefix.AvailableIPs(), int64(256))
	require.Equal(t, prefix.AcquiredIPs(), 3)

	err = ipam.ReleaseIP(*ip2)
	require.Nil(t, err)
	require.Equal(t, prefix.AvailableIPs(), int64(256))
	require.Equal(t, prefix.AcquiredIPs(), 2)

}

func TestIpamer_AcquireChildPrefix(t *testing.T) {
	type fields struct {
		storage Storage
		prefix  string
		length  int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Acquire next Prefix regularly",
			fields: fields{
				storage: memory{},
				prefix:  "192.168.0.0/31",
				length:  32,
			},
			want: "192.168.0.0/32",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Ipamer{
				storage: tt.fields.storage,
			}
			p, err := i.NewPrefix(tt.fields.prefix)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			got, err := i.AcquireChildPrefix(p, tt.fields.length)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			if !reflect.DeepEqual(got.Cidr, tt.want) {
				t.Errorf("Ipamer.AcquireChildPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrefix_AvailableIPs(t *testing.T) {
	tests := []struct {
		name string
		Cidr string
		want int64
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
		// {
		// 	name: "small IPv6",
		// 	Cidr: "2001:16b8:2d6a:6900:48d2:14a3:80ae:e797/64",
		// 	want: 4,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr: tt.Cidr,
			}
			if got := p.AvailableIPs(); got != tt.want {
				t.Errorf("Prefix.AvailableIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}
