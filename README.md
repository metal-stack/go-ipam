# go-ipam

[![Build Status](https://travis-ci.org/metal-pod/go-ipam.svg?branch=master)](https://travis-ci.org/metal-pod/go-ipam)
[![GoDoc](https://godoc.org/github.com/metal-pod/go-ipam?status.svg)](https://godoc.org/github.com/metal-pod/go-ipam)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-pod/go-ipam)](https://goreportcard.com/report/github.com/metal-pod/go-ipam)
[![codecov](https://codecov.io/gh/metal-pod/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-pod/go-ipam)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/metal-pod/go-ipam/blob/master/LICENSE)

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
    goipam "github.com/metal-pod/go-ipam"
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
    fmt.Printf("got IP: %s", ip.IP)

    prefix, err = ipam.ReleaseIP(ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.", ip.IP)
}
```

## Performance

```bash
BenchmarkNewPrefixMemory-4               1000000              1637 ns/op             728 B/op         27 allocs/op
BenchmarkNewPrefixPostgres-4                 200           8611579 ns/op            6170 B/op        155 allocs/op
BenchmarkAcquireIPMemory-4               1000000              1234 ns/op             232 B/op         15 allocs/op
BenchmarkAcquireIPPostgres-4                 200          11583345 ns/op            7252 B/op        184 allocs/op
BenchmarkAcquireChildPrefix1-4            300000              3771 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix2-4            300000              3773 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix3-4            300000              3997 ns/op            1541 B/op         58 allocs/op
BenchmarkAcquireChildPrefix4-4            300000              4877 ns/op            1581 B/op         60 allocs/op
BenchmarkAcquireChildPrefix5-4            200000              5541 ns/op            1854 B/op         70 allocs/op
BenchmarkAcquireChildPrefix6-4            300000              4123 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix7-4            300000              4954 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix8-4            300000              5017 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix9-4            300000              5309 ns/op            1528 B/op         58 allocs/op
BenchmarkAcquireChildPrefix10-4           200000              5234 ns/op            1532 B/op         58 allocs/op
BenchmarkPrefixOverlapping-4             1000000              1934 ns/op             432 B/op         24 allocs/op
```
