// https://networkengineering.stackexchange.com/questions/7106/how-do-you-calculate-the-prefix-network-subnet-and-host-numbers
//
// http://www.oznetnerd.com/subnetting-made-easy-part-1/

package ipam

import (
	"fmt"
	"net"
	"reflect"
	"testing"
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
				p.IPs[ipString] = IP{IP: i}
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

func TestIpamer_AcquireChildPrefix(t *testing.T) {
	type fields struct {
		storage Storage
		prefix  string
		length  int
	}
	tests := []struct {
		name   string
		fields fields
		want   *Prefix
	}{
		{
			name: "Acquire next IP regularly",
			fields: fields{
				storage: memory{},
				prefix:  "192.168.1.0/20",
				length:  22,
			},
			want: &Prefix{},
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ipamer.AcquireChildPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
