package ipam

import (
	"fmt"
	"testing"
)

func BenchmarkNewPrefix(b *testing.B) {
	_, pg, err := startPostgres()
	if err != nil {
		panic(err)
	}
	defer pg.db.Close()
	pgipam := NewWithStorage(pg)
	_, cock, err := startCockroach()
	if err != nil {
		panic(err)
	}
	defer cock.db.Close()
	cockipam := NewWithStorage(cock)
	benchmarks := []struct {
		name string
		ipam Ipamer
	}{
		{name: "Memory", ipam: New()},
		{name: "Postgres", ipam: pgipam},
		{name: "Cockroach", ipam: cockipam},
	}
	for _, bm := range benchmarks {
		test := bm
		b.Run(test.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p, err := test.ipam.NewPrefix("192.168.0.0/24")
				if err != nil {
					panic(err)
				}
				if p == nil {
					panic("Prefix nil")
				}
				_, err = test.ipam.DeletePrefix(p.Cidr)
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func BenchmarkAcquireIP(b *testing.B) {
	_, pg, err := startPostgres()
	if err != nil {
		panic(err)
	}
	defer pg.db.Close()
	pgipam := NewWithStorage(pg)
	_, cr, err := startCockroach()
	if err != nil {
		panic(err)
	}
	defer cr.db.Close()
	cockipam := NewWithStorage(cr)
	benchmarks := []struct {
		name string
		ipam Ipamer
		cidr string
	}{
		{name: "Memory", ipam: New(), cidr: "11.0.0.0/24"},
		{name: "Postgres", ipam: pgipam, cidr: "10.0.0.0/16"},
		{name: "Cockroach", ipam: cockipam, cidr: "10.0.0.0/16"},
	}
	for _, bm := range benchmarks {
		test := bm
		b.Run(test.name, func(b *testing.B) {
			p, err := test.ipam.NewPrefix(test.cidr)
			if err != nil {
				panic(err)
			}
			for n := 0; n < b.N; n++ {
				ip, err := test.ipam.AcquireIP(p.Cidr)
				if err != nil {
					panic(err)
				}
				if ip == nil {
					panic("IP nil")
				}
				p, err = test.ipam.ReleaseIP(ip)
				if err != nil {
					panic(err)
				}
			}
			_, err = test.ipam.DeletePrefix(test.cidr)
			if err != nil {
				b.Fatalf("error deleting prefix:%v", err)
			}
		})
	}
}

func BenchmarkAcquireChildPrefix(b *testing.B) {
	benchmarks := []struct {
		name         string
		parentLength uint8
		childLength  uint8
	}{
		{name: "8/14", parentLength: 8, childLength: 14},
		{name: "8/16", parentLength: 8, childLength: 16},
		{name: "8/20", parentLength: 8, childLength: 20},
		{name: "8/22", parentLength: 8, childLength: 22},
		{name: "8/24", parentLength: 8, childLength: 24},
		{name: "16/18", parentLength: 16, childLength: 18},
		{name: "16/20", parentLength: 16, childLength: 20},
		{name: "16/22", parentLength: 16, childLength: 22},
		{name: "16/24", parentLength: 16, childLength: 24},
		{name: "16/26", parentLength: 16, childLength: 26},
	}
	for _, bm := range benchmarks {
		test := bm
		b.Run(test.name, func(b *testing.B) {
			ipam := New()
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
