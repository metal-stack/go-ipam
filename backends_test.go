package ipam_test

// Import the backend providers for their side effect of registering all
// supported Storage backends with the shared test registry. This lives in an
// external test package so that the internal ipam tests can iterate over all
// backends without the core package depending on any database driver.
import (
	_ "github.com/metal-stack/go-ipam/pkg/test/backends"
)
