package ipam

// Ipamer can be used to do IPAM stuff.
type Ipamer interface {
	// NewPrefix create a new Prefix from a string notation.
	NewPrefix(cidr string) (*Prefix, error)
	// DeletePrefix delete a Prefix from a string notation.
	// If the Prefix is not found an NotFoundError is returned.
	DeletePrefix(cidr string) (*Prefix, error)
	// AcquireChildPrefix will return a Prefix with a smaller length from the given Prefix.
	AcquireChildPrefix(parentCidr string, length int) (*Prefix, error)
	// ReleaseChildPrefix will mark this child Prefix as available again.
	ReleaseChildPrefix(child *Prefix) error
	// PrefixFrom will return a known Prefix.
	PrefixFrom(cidr string) *Prefix
	// AcquireSpecificIP will acquire given IP and mark this IP as used, if already in use, return nil.
	// If specificIP is empty, the next free IP is returned.
	// If there is no free IP an NoIPAvailableError is returned.
	AcquireSpecificIP(prefixCidr, specificIP string) (*IP, error)
	// AcquireIP will return the next unused IP from this Prefix.
	AcquireIP(prefixCidr string) (*IP, error)
	// ReleaseIP will release the given IP for later usage and returns the updated Prefix.
	// If the IP is not found an NotFoundError is returned.
	ReleaseIP(ip *IP) (*Prefix, error)
	// ReleaseIPFromPrefix will release the given IP for later usage.
	// If the Prefix or the IP is not found an NotFoundError is returned.
	ReleaseIPFromPrefix(prefixCidr, ip string) error
	// PrefixesOverlapping will check if one ore more prefix of newPrefixes is overlapping
	// with one of existingPrefixes
	PrefixesOverlapping(existingPrefixes []string, newPrefixes []string) error
}

type ipamer struct {
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
