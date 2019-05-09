package ipam

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewWithStorage(t *testing.T) {
	storage := NewMemory()
	ipamer := NewWithStorage(storage)
	require.NotNil(t, ipamer)
	require.Equal(t, storage, ipamer.storage)
}

func Test_New(t *testing.T) {
	ipamer := New()
	require.NotNil(t, ipamer)
	require.NotNil(t, ipamer.storage)
}

func ExampleIpamer_NewPrefix() {
	ipamer := New()
	prefix, err := ipamer.NewPrefix("192.168.0.0/24")
	if err != nil {
		panic(err)
	}
	ip1, err := ipamer.AcquireIP(prefix)
	if err != nil {
		panic(err)
	}
	ip2, err := ipamer.AcquireIP(prefix)
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

	err = ipamer.ReleaseIP(ip2)
	if err != nil {
		panic(err)
	}

	err = ipamer.ReleaseIP(ip1)
	if err != nil {
		panic(err)
	}
	_, err = ipamer.DeletePrefix(prefix.Cidr)
	if err != nil {
		panic(err)
	}

}
