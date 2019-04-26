package ipam

import (
	"sync"
)

// Ipamer can be used to do IPAM stuff.
type Ipamer struct {
	storage Storage
}

// Network is a collection of prefixes, not sharing a common prefix length.
type Network struct {
	sync.Mutex
	ID       string
	Prefixes []Prefix
}

// New returns a Ipamer with in memory storage for networks, prefixes and ips.
func New() *Ipamer {
	return &Ipamer{storage: memory{}}
}

// NewWithStorage allows you to create a Ipamer instance with your Storage implementation.
// The Storage interface must be implemented.
func NewWithStorage(storage Storage) *Ipamer {
	return &Ipamer{storage: storage}
}

// FIXME locking
