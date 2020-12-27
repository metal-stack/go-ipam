package ipam

import (
	"reflect"
	"testing"

	"inet.af/netaddr"
)

func Test_extractPrefixFromSet(t *testing.T) {
	var (
		seta, setb netaddr.IPSet
	)
	pfx := netaddr.MustParseIPPrefix

	prefixa := pfx("192.168.0.0/16")
	seta.AddPrefix(prefixa)

	prefixb := pfx("192.168.128.0/18")
	setb.AddPrefix(prefixa)
	setb.RemovePrefix(prefixb)

	tests := []struct {
		name   string
		set    netaddr.IPSet
		length uint8
		want   netaddr.IPPrefix
		wantOK bool
	}{
		{
			name:   "simple",
			set:    seta,
			length: 24,
			want:   pfx("192.168.0.0/24"),
			wantOK: true,
		},
		{
			name:   "exact this",
			set:    seta,
			length: 16,
			want:   pfx("192.168.0.0/16"),
			wantOK: true,
		},
		{
			name:   "smaller",
			set:    seta,
			length: 18,
			want:   pfx("192.168.0.0/18"),
			wantOK: true,
		},
		{
			name:   "next",
			set:    setb,
			length: 18,
			want:   pfx("192.168.192.0/18"),
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPrefixFromSet(tt.set, tt.length)
			if ok != tt.wantOK {
				t.Errorf("extractPrefixFromSet() ok = %t, wantOK %t", ok, tt.wantOK)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractPrefixFromSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
