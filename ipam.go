package ipam

import (
	"context"
	"sync"
)

// Ipamer can be used to do IPAM stuff.
type Ipamer interface {
	// NewPrefix create a new Prefix from a string notation.
	NewPrefix(ctx context.Context, cidr string) (*Prefix, error)
	// DeletePrefix delete a Prefix from a string notation.
	// If the Prefix is not found an NotFoundError is returned.
	DeletePrefix(ctx context.Context, cidr string) (*Prefix, error)
	// AcquireChildPrefix will return a Prefix with a smaller length from the given Prefix.
	AcquireChildPrefix(ctx context.Context, parentCidr string, length uint8) (*Prefix, error)
	// AcquireSpecificChildPrefix will return a Prefix with a smaller length from the given Prefix.
	AcquireSpecificChildPrefix(ctx context.Context, parentCidr, childCidr string) (*Prefix, error)
	// ReleaseChildPrefix will mark this child Prefix as available again.
	ReleaseChildPrefix(ctx context.Context, child *Prefix) error
	// PrefixFrom will return a known Prefix.
	PrefixFrom(ctx context.Context, cidr string) *Prefix
	// AcquireSpecificIP will acquire given IP and mark this IP as used, if already in use, return nil.
	// If specificIP is empty, the next free IP is returned.
	// If there is no free IP an NoIPAvailableError is returned.
	AcquireSpecificIP(ctx context.Context, prefixCidr, specificIP string) (*IP, error)
	// AcquireIP will return the next unused IP from this Prefix.
	AcquireIP(ctx context.Context, prefixCidr string) (*IP, error)
	// ReleaseIP will release the given IP for later usage and returns the updated Prefix.
	// If the IP is not found an NotFoundError is returned.
	ReleaseIP(ctx context.Context, ip *IP) (*Prefix, error)
	// ReleaseIPFromPrefix will release the given IP for later usage.
	// If the Prefix or the IP is not found an NotFoundError is returned.
	ReleaseIPFromPrefix(ctx context.Context, prefixCidr, ip string) error
	// Dump all stored prefixes as json formatted string
	Dump(ctx context.Context) (string, error)
	// Load a previously created json formatted dump, deletes all prefixes before loading
	Load(ctx context.Context, dump string) error
	// ReadAllPrefixCidrs retrieves all existing Prefix CIDRs from the underlying storage
	ReadAllPrefixCidrs(ctx context.Context) ([]string, error)
}

type ipamer struct {
	mu      sync.Mutex
	storage Storage
}

// New returns a Ipamer with in memory storage for networks, prefixes and ips.
func New() Ipamer {
	storage := NewMemory()
	return &ipamer{storage: storage}
}

// NewWithStorage allows you to create a Ipamer instance with your Storage implementation.
// The Storage interface must be implemented.
func NewWithStorage(storage Storage) Ipamer {
	return &ipamer{storage: storage}
}
