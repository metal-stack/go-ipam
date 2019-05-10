package ipam

import (
	"fmt"
	"testing"
)

func benchmarkNewPrefix(ipam *Ipamer, b *testing.B) {
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
	storage, _ := NewPostgresStorage("localhost", "5433", "postgres", "password", "postgres", "disable")
	ipam := NewWithStorage(storage)
	benchmarkNewPrefix(ipam, b)
}

func benchmarkAquireIP(ipam *Ipamer, b *testing.B) {
	p, err := ipam.NewPrefix("10.0.0.0/24")
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		ip, err := ipam.AcquireIP(p)
		if err != nil {
			panic(err)
		}
		if ip == nil {
			panic("IP nil")
		}
		err = ipam.ReleaseIP(ip)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkAquireIPMemory(b *testing.B) {
	ipam := New()
	benchmarkAquireIP(ipam, b)
}
func BenchmarkAquireIPPostgres(b *testing.B) {
	storage, _ := NewPostgresStorage("localhost", "5433", "postgres", "password", "postgres", "disable")
	ipam := NewWithStorage(storage)
	benchmarkAquireIP(ipam, b)
}

func benchmarkAquireChildPrefix(parentLength, childLength int, b *testing.B) {
	ipam := New()
	p, err := ipam.NewPrefix(fmt.Sprintf("192.168.0.0/%d", parentLength))
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		p, err := ipam.AcquireChildPrefix(p, childLength)
		if err != nil {
			panic(err)
		}
		err = ipam.ReleaseChildPrefix(p)
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkAquireChildPrefix1(b *testing.B)  { benchmarkAquireChildPrefix(8, 14, b) }
func BenchmarkAquireChildPrefix2(b *testing.B)  { benchmarkAquireChildPrefix(8, 16, b) }
func BenchmarkAquireChildPrefix3(b *testing.B)  { benchmarkAquireChildPrefix(8, 20, b) }
func BenchmarkAquireChildPrefix4(b *testing.B)  { benchmarkAquireChildPrefix(8, 22, b) }
func BenchmarkAquireChildPrefix5(b *testing.B)  { benchmarkAquireChildPrefix(8, 24, b) }
func BenchmarkAquireChildPrefix6(b *testing.B)  { benchmarkAquireChildPrefix(16, 18, b) }
func BenchmarkAquireChildPrefix7(b *testing.B)  { benchmarkAquireChildPrefix(16, 20, b) }
func BenchmarkAquireChildPrefix8(b *testing.B)  { benchmarkAquireChildPrefix(16, 22, b) }
func BenchmarkAquireChildPrefix9(b *testing.B)  { benchmarkAquireChildPrefix(16, 24, b) }
func BenchmarkAquireChildPrefix10(b *testing.B) { benchmarkAquireChildPrefix(16, 26, b) }
