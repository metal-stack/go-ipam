# go-ipam

[![Actions](https://github.com/metal-pod/go-ipam/workflows/build/badge.svg)](https://github.com/metal-pod/go-ipam/actions)
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

## CockroachDB

There are currently test undergoing, make this code work as well with cockroachdb, this works with no code modifications except for the concurrent access.
If running `make test` the following error occurs:

```bash
--- FAIL: Test_ConcurrentAcquirePrefix (7.07s)
    sql_test.go:216: unable to update parent prefix:1.0.0.0/16: pq: restart transaction: TransactionRetryWithProtoRefreshError: TransactionAbortedError(ABORT_REASON_PUSHER_ABORTED): "sql txn" id=3cdd3c6c key=/Ta
ble/78/1/"1.0.0.0/16"/0 rw=true pri=0.01752018 stat=ABORTED epo=0 ts=1578921623.657150405,1 orig=1578921623.654689427,0 min=1578921623.654689427,0 max=1578921624.154689427,0 wto=false seq=1
    sql_test.go:216: unable to update parent prefix:1.0.0.0/16: pq: restart transaction: TransactionRetryWithProtoRefreshError: TransactionAbortedError(ABORT_REASON_PUSHER_ABORTED): "sql txn" id=2f9fd074 key=/Ta
ble/78/1/"1.0.0.0/16"/0 rw=true pri=0.02781474 stat=ABORTED epo=0 ts=1578921623.657150405,1 orig=1578921623.653808793,0 min=1578921623.653808793,0 max=1578921624.153808793,0 wto=false seq=1
    sql_test.go:216: unable to update parent prefix:1.0.0.0/16: pq: restart transaction: TransactionRetryWithProtoRefreshError: TransactionAbortedError(ABORT_REASON_PUSHER_ABORTED): "sql txn" id=fc520ec2 key=/Ta
ble/78/1/"1.0.0.0/16"/0 rw=true pri=0.00422574 stat=ABORTED epo=0 ts=1578921623.669968618,1 orig=1578921623.650526792,0 min=1578921623.650526792,0 max=1578921624.150526792,0 wto=false seq=1
    sql_test.go:216: unable to update parent prefix:1.0.0.0/16: pq: restart transaction: TransactionRetryWithProtoRefreshError: WriteTooOldError: write at timestamp 1578921623.664788862,0 too old; wrote at 15789
21623.913407511,1
    sql_test.go:216: unable to update parent prefix:1.0.0.0/16: pq: restart transaction: TransactionRetryWithProtoRefreshError: WriteTooOldError: write at timestamp 1578921623.669392554,0 too old; wrote at 15789
21623.913407511,1
```

I could not find any matching issue on the cockroachdb issue tracker thought.
