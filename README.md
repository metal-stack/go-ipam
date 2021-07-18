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

## Supported Databases

|                              | Postgres | CockroachDB | Redis   | KeyDB   | Memory     |
|------------------------------|----------|-------------|---------|---------|------------|
| Production ready             | Y        | Y           | Y       | Y       | N          |
| geo redundant setup possible | N        | Y           | N       | Y       | N          |
| AcquireIP/sec                | ~100/s   | ~60/s       | ~1400/s | ~1400/s | >200.000/s |
| AcquireChildPrefix/sec       | ~40/s    | ~35/s       | ~1000/s | ~1000/s | >100.000/s |

Test were run on a Intel(R) Core(TM) i5-6600 CPU @ 3.30GHz

## Performance

```bash
BenchmarkNewPrefix/Memory-4               464994        2675 ns/op     1728 B/op     27 allocs/op
BenchmarkNewPrefix/Postgres-4                126    11775448 ns/op     6259 B/op    144 allocs/op
BenchmarkNewPrefix/Cockroach-4               100    25558820 ns/op     6250 B/op    144 allocs/op
BenchmarkNewPrefix/Redis-4                  3854      308122 ns/op     3930 B/op     78 allocs/op
BenchmarkNewPrefix/KeyDB-4                  3907      307655 ns/op     3930 B/op     78 allocs/op
BenchmarkAcquireIP/Memory-4               229524        4508 ns/op     2680 B/op     56 allocs/op
BenchmarkAcquireIP/Postgres-4                 98    14918027 ns/op    10684 B/op    263 allocs/op
BenchmarkAcquireIP/Cockroach-4                51    19688920 ns/op    10728 B/op    264 allocs/op
BenchmarkAcquireIP/Redis-4                  1734      695545 ns/op    12113 B/op    268 allocs/op
BenchmarkAcquireIP/KeyDB-4                  1476      751854 ns/op    12110 B/op    268 allocs/op
BenchmarkAcquireChildPrefix/Memory-4      128704        8453 ns/op     5201 B/op     94 allocs/op
BenchmarkAcquireChildPrefix/Postgres-4        70    21220704 ns/op    15663 B/op    378 allocs/op
BenchmarkAcquireChildPrefix/Cockroach-4       32    37638608 ns/op    15774 B/op    381 allocs/op
BenchmarkAcquireChildPrefix/Redis-4         1280      925054 ns/op    16016 B/op    349 allocs/op
BenchmarkAcquireChildPrefix/KeyDB-4         1143      953056 ns/op    16018 B/op    349 allocs/op
BenchmarkPrefixOverlapping-4             4306106       274.4 ns/op        0 B/op      0 allocs/op
```

## Testing individual Backends

It is possible to test a individual backend only to speed up development roundtrip.

`backend` can be one of `Memory`, `Postgres`, `Cockroach` and `Redis`.

```bash
BACKEND=backend make test
```
