package ipam

import (
	"fmt"
)

func ExampleIpamer_NewPrefix() {
	ipamer := New()
	prefix, err := ipamer.NewPrefix("192.168.0.0/24")
	if err != nil {
		panic(err)
	}
	ip1, err := ipamer.AcquireIP(prefix.Cidr)
	if err != nil {
		panic(err)
	}
	ip2, err := ipamer.AcquireIP(prefix.Cidr)
	if err != nil {
		panic(err)
	}

	fmt.Println(prefix)
	fmt.Println(ip1.IP.String())
	fmt.Println(ip1.ParentPrefix)
	fmt.Println(ip2.IP.String())
	fmt.Println(ip2.ParentPrefix)
	// Output:
	// 192.168.0.0/24
	// 192.168.0.1
	// 192.168.0.0/24
	// 192.168.0.2
	// 192.168.0.0/24

	_, err = ipamer.ReleaseIP(ip2)
	if err != nil {
		panic(err)
	}

	_, err = ipamer.ReleaseIP(ip1)
	if err != nil {
		panic(err)
	}
	_, err = ipamer.DeletePrefix(prefix.Cidr)
	if err != nil {
		panic(err)
	}

}
