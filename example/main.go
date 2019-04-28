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

	n, err := i.NewNetwork()
	if err != nil {
		log.Fatal(err)
	}

	storedNetworks, err := pgStorage.ReadAllNetworks()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("network:%v created:%v", n, storedNetworks)

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

	p0 := i.PrefixFrom("10.0.2.0/25")
	log.Printf("found prefix:%v", p0)

	p1, _ := i.NewPrefix("1.2.1.0/24")
	p2, _ := i.NewPrefix("1.2.2.0/24")
	p3, _ := i.NewPrefix("1.2.3.0/24")
	// p4, _ := i.NewPrefix("1.2.4.0/24")
	// p5, _ := i.NewPrefix("1.2.5.0/24")
	p6, _ := i.NewPrefix("1.2.0.0/22")
	_, err = i.NewNetwork(*p1, *p2, *p3)
	if err != nil {
		log.Fatal(err)
	}
	n1, err := i.NewNetwork()
	if err != nil {
		log.Fatal(err)
	}
	_, err = i.AddPrefix(n1, p6)
	if err != nil {
		log.Fatal(err)
	}

}
