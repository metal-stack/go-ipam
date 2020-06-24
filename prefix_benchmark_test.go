package ipam

import (
	"fmt"
	"testing"
)

func benchmarkNewPrefix(ipam Ipamer, b *testing.B) {
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
}
func BenchmarkNewPrefixMemory(b *testing.B) {
	ipam := New()
	benchmarkNewPrefix(ipam, b)
}
func BenchmarkNewPrefixPostgres(b *testing.B) {
	_, storage, err := startPostgres()
	if err != nil {
		panic(err)
	}
	defer storage.db.Close()
	ipam := NewWithStorage(storage)
	benchmarkNewPrefix(ipam, b)
}
func BenchmarkNewPrefixCockroach(b *testing.B) {
	_, storage, err := startCockroach()
	if err != nil {
		panic(err)
	}
	defer storage.db.Close()
	ipam := NewWithStorage(storage)
	benchmarkNewPrefix(ipam, b)
}

func benchmarkAcquireIP(ipam Ipamer, cidr string, b *testing.B) {
	p, err := ipam.NewPrefix(cidr)
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
	_, err = ipam.DeletePrefix(cidr)
	if err != nil {
		b.Fatalf("error deleting prefix:%v", err)
	}
}

func BenchmarkAcquireIPMemory(b *testing.B) {
	ipam := New()
	benchmarkAcquireIP(ipam, "11.0.0.0/24", b)
}
func BenchmarkAcquireIPPostgres(b *testing.B) {
	_, storage, err := startPostgres()
	if err != nil {
		panic(err)
	}
	defer storage.db.Close()
	ipam := NewWithStorage(storage)
	benchmarkAcquireIP(ipam, "10.0.0.0/16", b)
}

func BenchmarkAcquireIPCockroach(b *testing.B) {
	_, storage, err := startCockroach()
	if err != nil {
		panic(err)
	}
	defer storage.db.Close()
	ipam := NewWithStorage(storage)
	benchmarkAcquireIP(ipam, "10.0.0.0/16", b)
}

func benchmarkAcquireChildPrefix(parentLength, childLength int, b *testing.B) {
	ipam := New()
	p, err := ipam.NewPrefix(fmt.Sprintf("192.168.0.0/%d", parentLength))
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		p, err := ipam.AcquireChildPrefix(p.Cidr, childLength)
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
}
func BenchmarkAcquireChildPrefix1(b *testing.B)  { benchmarkAcquireChildPrefix(8, 14, b) }
func BenchmarkAcquireChildPrefix2(b *testing.B)  { benchmarkAcquireChildPrefix(8, 16, b) }
func BenchmarkAcquireChildPrefix3(b *testing.B)  { benchmarkAcquireChildPrefix(8, 20, b) }
func BenchmarkAcquireChildPrefix4(b *testing.B)  { benchmarkAcquireChildPrefix(8, 22, b) }
func BenchmarkAcquireChildPrefix5(b *testing.B)  { benchmarkAcquireChildPrefix(8, 24, b) }
func BenchmarkAcquireChildPrefix6(b *testing.B)  { benchmarkAcquireChildPrefix(16, 18, b) }
func BenchmarkAcquireChildPrefix7(b *testing.B)  { benchmarkAcquireChildPrefix(16, 20, b) }
func BenchmarkAcquireChildPrefix8(b *testing.B)  { benchmarkAcquireChildPrefix(16, 22, b) }
func BenchmarkAcquireChildPrefix9(b *testing.B)  { benchmarkAcquireChildPrefix(16, 24, b) }
func BenchmarkAcquireChildPrefix10(b *testing.B) { benchmarkAcquireChildPrefix(16, 26, b) }

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
