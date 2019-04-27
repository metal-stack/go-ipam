package ipam

// Ipamer can be used to do IPAM stuff.
type Ipamer struct {
	storage Storage
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
