# go-ipam

[![Actions](https://github.com/metal-stack/go-ipam/workflows/build/badge.svg)](https://github.com/metal-stack/go-ipam/actions)
[![GoDoc](https://godoc.org/github.com/metal-stack/go-ipam?status.svg)](https://godoc.org/github.com/metal-stack/go-ipam)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-stack/go-ipam)](https://goreportcard.com/report/github.com/metal-stack/go-ipam)
[![codecov](https://codecov.io/gh/metal-stack/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-stack/go-ipam)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/metal-stack/go-ipam/blob/master/LICENSE)

go-ipam is a module to handle IP address management. It can operate on networks, prefixes and IPs.

## IP

Most obvious this library is all about IP management. The main purpose is to acquire and release an IP, or a bunch of
IP's from prefixes.

## Prefix

A prefix is a network with IP and mask, typically in the form of *192.168.0.0/24*. To be able to manage IPs you have to create a prefix first.

Example usage:

```go

package main

import (
    "fmt"
    goipam "github.com/metal-stack/go-ipam"
)

func main() {
    // create a ipamer with in memory storage
    ipam := goipam.New()

    prefix, err := ipam.NewPrefix("192.168.0.0/24")
    if err != nil {
        panic(err)
    }

    ip, err := ipam.AcquireIP(prefix.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s\n", ip.IP)

    prefix, err = ipam.ReleaseIP(ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.\n", ip.IP)

    // Now a IPv6 Super Prefix with Child Prefixes
    prefix, err = ipam.NewPrefix("2001:aabb::/48")
    if err != nil {
        panic(err)
    }
    cp1, err := ipam.AcquireChildPrefix(prefix.Cidr, 64)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got Prefix: %s\n", cp1)
    cp2, err := ipam.AcquireChildPrefix(prefix.Cidr, 72)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got Prefix: %s\n", cp2)
    ip21, err := ipam.AcquireIP(cp2.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s\n", ip21.IP)
}
```

## Supported Databases & Performance

| Database     |  Acquire Child Prefix |  Acquire IP |   New Prefix |  Prefix Overlap | Production-Ready | Geo-Redundant          |
|:-------------|----------------------:|------------:|-------------:|----------------:|:-----------------|:-----------------------|
| In-Memory    |           106,861/sec | 196,687/sec |  330,578/sec |         248/sec | N                | N                      |
| KeyDB        |               777/sec |     975/sec |    2,271/sec |                 | Y                | Y                      |
| Redis        |               773/sec |     958/sec |    2,349/sec |                 | Y                | N                      |
| MongoDB      |               415/sec |     682/sec |      772/sec |                 | Y                | Y                      |
| Etcd         |               258/sec |     368/sec |      533/sec |                 | Y                | N                      |
| Postgres     |               203/sec |     331/sec |      472/sec |                 | Y                | N                      |
| CockroachDB  |                40/sec |      37/sec |       46/sec |                 | Y                | Y                      |

The benchmarks above were performed using:
 * cpu: Intel(R) Xeon(R) Platinum 8370C CPU @ 2.80GHz
 * postgres:14-alpine 
 * cockroach:v22.1.0 
 * redis:7.0-alpine 
 * keydb:alpine_x86_64_v6.2.2
 * etcd:v3.5.4 
 * mongodb:5.0.9-focal

### Database Version Compatability
| Database    | Details                                                                                       |
|-------------|-----------------------------------------------------------------------------------------------|
| KeyDB       |                                                                                               |
| Redis       |                                                                                               |
| MongoDB     | https://www.mongodb.com/docs/drivers/go/current/compatibility/#std-label-golang-compatibility |
| Etcd        |                                                                                               |
| Postgres    |                                                                                               |
| CockroachDB |                                                                                               |

## Testing individual Backends

It is possible to test a individual backend only to speed up development roundtrip.

`backend` can be one of `Memory`, `Postgres`, `Cockroach`, `Etcd`, `Redis`, and `MongoDB`.

```bash
BACKEND=backend make test
```
