package ipam

import "fmt"

// Ipamer can be used to do IPAM stuff.
type Ipamer struct {
	storage Storage
}

// New returns a Ipamer with in memory storage for networks, prefixes and ips.
func New() *Ipamer {
	storage := NewMemory()
	return &Ipamer{storage: storage}
}

// NewWithStorage allows you to create a Ipamer instance with your Storage implementation.
// The Storage interface must be implemented.
func NewWithStorage(storage Storage) *Ipamer {
	return &Ipamer{storage: storage}
}

// NotFoundError is raised if the given Prefix or Cidr was not found
type NotFoundError struct {
	msg string
}

func (n NotFoundError) Error() string {
	return n.msg
}

func newNotFoundError(msg string) NotFoundError {
	return NotFoundError{msg: msg}
}

func newNotFoundErrorf(msg string, a ...interface{}) NotFoundError {
	return NotFoundError{msg: fmt.Sprintf(msg, a...)}
}
