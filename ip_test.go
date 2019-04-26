package ipam

import (
	"net"
	"reflect"
	"testing"
)

func TestIP_lshift(t *testing.T) {
	tests := []struct {
		name string
		IP   net.IP
		bits uint8
		want IP
	}{
		{
			name: "test1",
			IP:   net.ParseIP("0.0.0.0"),
			bits: 2,
			want: IP{IP: net.ParseIP("1.2.3.2")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IP{
				IP: tt.IP,
			}
			if got := i.lshift(tt.bits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IP.lshift() = %v, want %v", got, tt.want)
			}
		})
	}
}
