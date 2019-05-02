package main

import (
	"log"

	ipam "github.com/metal-pod/go-ipam"
)

func main() {
	pgStorage, err := ipam.NewPostgresStorage("localhost", "5432", "postgres", "password", "postgres", "disable")

	if err != nil {
		log.Fatal(err)
	}

	i := ipam.NewWithStorage(pgStorage)

	_, err = i.NewPrefix("10.0.0.0/16")
	if err != nil {
		log.Fatal(err)
	}
	p := i.PrefixFrom("10.0.0.0/16")
	c, err := i.AcquireChildPrefix(p, 22)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("child:%s\n", c.Cidr)

	for index := 0; index < 60; index++ {
		c, err := i.AcquireChildPrefix(p, 22)
		if err != nil {
			log.Fatal(err)
		}
		// for {
		ip, err := i.AcquireIP(c)
		if err != nil {
			log.Fatal(err)
		}

		if ip == nil {
			break
		}
		// log.Printf("ip %s created in %s", ip.IP, c.Cidr)
		// }

		log.Printf("child prefix created:%v", c.Cidr)

	}

	p0 := i.PrefixFrom("10.0.2.0/22")
	log.Printf("found prefix:%v", p0)

	i.NewPrefix("1.2.1.0/24")
	i.NewPrefix("1.2.2.0/24")
	i.NewPrefix("1.2.3.0/24")
	i.NewPrefix("1.2.0.0/22")

}
