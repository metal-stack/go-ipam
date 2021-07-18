package ipam

import (
	"fmt"
	"testing"
)

func BenchmarkNewPrefix(b *testing.B) {
	benchWithBackends(b, func(b *testing.B, ipam *ipamer) {
		for n := 0; n < b.N; n++ {
			p, err := ipam.NewPrefix("192.168.0.0/24")
			if err != nil {
				panic(err)
			}
			if p == nil {
				panic("Prefix nil")
			}
			_, err = ipam.DeletePrefix(p.Cidr)
			if err != nil {
				panic(err)
			}
		}
	})
}

func BenchmarkAcquireIP(b *testing.B) {
	testCidr := "10.0.0.0/16"
	benchWithBackends(b, func(b *testing.B, ipam *ipamer) {
		p, err := ipam.NewPrefix(testCidr)
		if err != nil {
			panic(err)
		}
		for n := 0; n < b.N; n++ {
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
		_, err = ipam.DeletePrefix(testCidr)
		if err != nil {
			b.Fatalf("error deleting prefix:%v", err)
		}
	})
}

func BenchmarkAcquireChildPrefix(b *testing.B) {
	benchmarks := []struct {
		name         string
		parentLength uint8
		childLength  uint8
	}{
		{name: "8/14", parentLength: 8, childLength: 14},
		// {name: "8/16", parentLength: 8, childLength: 16},
		// {name: "8/20", parentLength: 8, childLength: 20},
		// {name: "8/22", parentLength: 8, childLength: 22},
		// {name: "8/24", parentLength: 8, childLength: 24},
		// {name: "16/18", parentLength: 16, childLength: 18},
		// {name: "16/20", parentLength: 16, childLength: 20},
		// {name: "16/22", parentLength: 16, childLength: 22},
		// {name: "16/24", parentLength: 16, childLength: 24},
		// {name: "16/26", parentLength: 16, childLength: 26},
	}
	for _, bm := range benchmarks {
		test := bm
		benchWithBackends(b, func(b *testing.B, ipam *ipamer) {
			p, err := ipam.NewPrefix(fmt.Sprintf("192.168.0.0/%d", test.parentLength))
			if err != nil {
				panic(err)
			}
			for n := 0; n < b.N; n++ {
				p, err := ipam.AcquireChildPrefix(p.Cidr, test.childLength)
				if err != nil {
					panic(err)
				}
				err = ipam.ReleaseChildPrefix(p)
				if err != nil {
					panic(err)
				}
			}
			_, err = ipam.DeletePrefix(p.Cidr)
			if err != nil {
				b.Fatalf("error deleting prefix:%v", err)
			}
		})
	}
}

func BenchmarkPrefixOverlapping(b *testing.B) {
	ipam := New()
	existingPrefixes := []string{"192.168.0.0/24", "10.0.0.0/8"}
	newPrefixes := []string{"192.168.1.0/24", "11.0.0.0/8"}
	for n := 0; n < b.N; n++ {
		err := ipam.PrefixesOverlapping(existingPrefixes, newPrefixes)
		if err != nil {
			b.Errorf("PrefixOverLapping error:%v", err)
		}
	}
}
