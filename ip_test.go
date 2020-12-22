package ipam

import (
	"testing"

	"inet.af/netaddr"
)

func TestPrefix(t *testing.T) {

	prefix, _ := netaddr.ParseIPPrefix("192.168.0.0/22")
	remove, _ := netaddr.ParseIPPrefix("192.168.0.0/28")
	remove2, _ := netaddr.ParseIPPrefix("192.168.1.0/28")

	var ipset netaddr.IPSet

	ipset.AddPrefix(prefix)
	ipset.RemovePrefix(remove)
	ipset.RemovePrefix(remove2)

	// var iprange netaddr.IPRange
	// iprange.Prefixes()
	length := uint8(28)

	for _, p := range ipset.Prefixes() {
		if p.Bits == length-1 {
			t.Logf("next available prefix:%s range:%s", p, p.Range())
			subrange := netaddr.IPRange{From: p.Range().From, To: p.Range().To.Prior()}
			nextChild := subrange.Prefixes()[0]
			t.Logf("childprefix:%s", nextChild)
			ipset.RemovePrefix(nextChild)
			break
		}
		t.Logf("prefix:%s", p)
	}
}

func TestIPAcquire(t *testing.T) {
	ipnet, _ := netaddr.ParseIPPrefix("192.168.0.0/28")

	from := ipnet.Range().From
	for ip := from; ipnet.Contains(ip); ip = ip.Next() {

		t.Logf("IP:%s", ip)
	}
}
