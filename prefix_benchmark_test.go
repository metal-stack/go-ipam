package ipam

import (
	"fmt"
	"testing"
)

func BenchmarkNewPrefix(b *testing.B) {
	ipam := New()
	for n := 0; n < b.N; n++ {
		p, err := ipam.NewPrefix("192.168.0.0/24")
		if err != nil {
			panic(err)
		}
		_, err = ipam.DeletePrefix(p.Cidr)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkAquireIP(b *testing.B) {
	ipam := New()
	p, err := ipam.NewPrefix("192.168.0.0/24")
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		ip, err := ipam.AcquireIP(p)
		if err != nil {
			panic(err)
		}
		err = ipam.ReleaseIP(ip)
		if err != nil {
			panic(err)
		}
	}
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
