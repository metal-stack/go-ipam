// Package testregistry provides a tiny global registry which allows test-only
// packages to register Storage providers without forcing the core ipam package
// to depend on any database driver.
//
// It intentionally does not import the ipam package (providers are stored as
// any) so that it can be imported by the ipam package's internal tests without
// creating an import cycle.
package testregistry

import "context"

// Provider describes a Storage backend which can be used in tests and
// benchmarks.
type Provider struct {
	// Name is the human readable name of the backend, e.g. "Postgres".
	Name string
	// Provide creates a fresh Storage instance.
	Provide func(ctx context.Context) (any, error)
	// Cleanup is called before a test/benchmark runs. It may be nil.
	Cleanup func(ctx context.Context, storage any) error
	// PostCleanup is called after a test/benchmark ran. It may be nil.
	PostCleanup func(ctx context.Context, storage any) error
}

var providers []Provider

// Register adds a provider to the global registry. It is expected to be called
// from an init function.
func Register(p Provider) {
	providers = append(providers, p)
}

// All returns all registered providers.
func All() []Provider {
	return providers
}
