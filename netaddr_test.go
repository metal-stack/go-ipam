package ipam

import (
	"reflect"
	"testing"

	"inet.af/netaddr"
)

func Test_extractPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  netaddr.IPPrefix
		length  uint8
		want    netaddr.IPPrefix
		wantErr bool
	}{
		{
			name:    "simple",
			prefix:  mustIPPrefix("192.168.0.0/16"),
			length:  24,
			want:    mustIPPrefix("192.168.0.0/24"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractPrefix(tt.prefix, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extractPrefixFromSet(t *testing.T) {

	var (
		seta, setb netaddr.IPSet
	)
	prefixa := mustIPPrefix("192.168.0.0/16")
	seta.AddPrefix(prefixa)

	prefixb := mustIPPrefix("192.168.128.0/18")
	setb.AddPrefix(prefixa)
	setb.RemovePrefix(prefixb)

	tests := []struct {
		name    string
		set     netaddr.IPSet
		length  uint8
		want    netaddr.IPPrefix
		wantErr bool
	}{
		{
			name:    "simple",
			set:     seta,
			length:  24,
			want:    mustIPPrefix("192.168.0.0/24"),
			wantErr: false,
		},
		{
			name:    "exact this",
			set:     seta,
			length:  16,
			want:    mustIPPrefix("192.168.0.0/16"),
			wantErr: false,
		},
		{
			name:    "smaller",
			set:     seta,
			length:  18,
			want:    mustIPPrefix("192.168.0.0/18"),
			wantErr: false,
		},
		{
			name:    "next",
			set:     setb,
			length:  18,
			want:    mustIPPrefix("192.168.192.0/18"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractPrefixFromSet(tt.set, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractPrefixFromSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractPrefixFromSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustIPPrefix(s string) netaddr.IPPrefix {
	p, err := netaddr.ParseIPPrefix(s)
	if err != nil {
		panic(err)
	}

	return p
}
