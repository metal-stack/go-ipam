/*
Package ipam is a ip address management library for ip's and prefixes (networks).

The core package only contains the pure-Go implementation and the in-memory
Storage. Every database backed Storage lives in its own package so that
consumers only pull in the dependencies of the backends they actually use:

  - github.com/metal-stack/go-ipam/pkg/postgres
  - github.com/metal-stack/go-ipam/pkg/redis
  - github.com/metal-stack/go-ipam/pkg/etcd
  - github.com/metal-stack/go-ipam/pkg/mongodb
  - github.com/metal-stack/go-ipam/pkg/file

You can also bring your own Storage implementation as you need.

Example usage:

	import (
		"fmt"
		goipam "github.com/metal-stack/go-ipam"
	)


	func main() {
		ctx := context.Background()
		// create a ipamer with in memory storage
		ipam := goipam.New(ctx)

		prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
		if err != nil {
			panic(err)
		}

		ip, err := ipam.AcquireIP(ctx, prefix.Cidr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("got IP: %s", ip.IP)

		err = ipam.ReleaseIP(ctx, ip)
		if err != nil {
			panic(err)
		}
		fmt.Printf("IP: %s released.", ip.IP)
	}
*/
package ipam
