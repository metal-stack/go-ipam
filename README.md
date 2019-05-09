# go-ipam

[![Build Status](https://travis-ci.org/metal-pod/go-ipam.svg?branch=master)](https://travis-ci.org/metal-pod/go-ipam)
[![GoDoc](https://godoc.org/github.com/metal-pod/go-ipam?status.svg)](https://godoc.org/github.com/metal-pod/go-ipam)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-pod/go-ipam)](https://goreportcard.com/report/github.com/metal-pod/go-ipam)
[![codecov](https://codecov.io/gh/metal-pod/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-pod/go-ipam)

go-ipam is a module to handle IPAddress management. It can operate on Networks, Prefixes and IPs.

## IP

Most obvious this library is all about ip management. the main purpose is to acquire and release an ip, or a bunch of
ip's from prefixes.

## Prefix

A prefix is a network with ip and mask, typicaly in the form of *192.168.0.0/24*. To be able to manage IPs you have to create a prefix first.

example usage:

```go

package main

import (
    "fmt"
    goipam "github.com/metal-pod/go-ipam"
)


func main() {
    // create a ipamer with in memory storage
    ipam := goipam.New()

    prefix, err := ipam.NewPrefix("192.168.0.0/24")
    if err != nil {
        panic(err)
    }

    ip, err := ipam.AcquireIP(prefix)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s", ip.IP)

    err = ipam.ReleaseIP(ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.", ip.IP)
}
```

## Performance

```bash
BenchmarkNewPrefix-4                     1000000              1554 ns/op
BenchmarkAquireIP-4                      1000000              1088 ns/op
BenchmarkAquireChildPrefix1-4             300000              3667 ns/op
BenchmarkAquireChildPrefix2-4             300000              3685 ns/op
BenchmarkAquireChildPrefix3-4             300000              3921 ns/op
BenchmarkAquireChildPrefix4-4             300000              4332 ns/op
BenchmarkAquireChildPrefix5-4             200000              5684 ns/op
BenchmarkAquireChildPrefix6-4             300000              3690 ns/op
BenchmarkAquireChildPrefix7-4             300000              3731 ns/op
BenchmarkAquireChildPrefix8-4             300000              4152 ns/op
BenchmarkAquireChildPrefix9-4             500000              3727 ns/op
BenchmarkAquireChildPrefix10-4            300000              3843 ns/op
```
