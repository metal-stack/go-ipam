package ipam

import (
	"fmt"
	"os"
	"testing"

	"github.com/metal-stack/go-ipam/pkg/testregistry"
)

// backend can be used to restrict tests/benchmarks to a single backend, e.g.
// BACKEND=Postgres go test ./...
var backend string

func TestMain(m *testing.M) {
	backend = os.Getenv("BACKEND")
	if backend == "" {
		fmt.Printf("Using all backends\n")
	} else {
		fmt.Printf("only test %s\n", backend)
	}
	os.Exit(m.Run())
}

type benchMethod func(b *testing.B, ipam *ipamer)

func benchWithBackends(b *testing.B, fn benchMethod) {
	for _, provider := range testregistry.All() {
		if backend != "" && backend != provider.Name {
			continue
		}
		raw, err := provider.Provide(b.Context())
		if err != nil {
			b.Fatalf("error providing %s storage: %v", provider.Name, err)
		}
		storage, ok := raw.(Storage)
		if !ok {
			b.Fatalf("provider %s did not return a Storage: %T", provider.Name, raw)
		}

		if provider.Cleanup != nil {
			if err := provider.Cleanup(b.Context(), raw); err != nil {
				b.Errorf("error cleaning up before the test: %v", err)
			}
		}

		ipamer := &ipamer{storage: storage}
		testName := provider.Name

		b.Run(testName, func(b *testing.B) {
			fn(b, ipamer)
		})

		if provider.PostCleanup != nil {
			if err := provider.PostCleanup(b.Context(), raw); err != nil {
				b.Errorf("error cleaning up after the test: %v", err)
			}
		}
	}
}

type testMethod func(t *testing.T, ipam *ipamer)

func testWithBackends(t *testing.T, fn testMethod) {
	t.Helper()
	// prevent testcontainer logging mangle test and benchmark output
	for _, provider := range testregistry.All() {
		if backend != "" && backend != provider.Name {
			continue
		}
		raw, err := provider.Provide(t.Context())
		if err != nil {
			t.Fatalf("error providing %s storage: %v", provider.Name, err)
		}
		storage, ok := raw.(Storage)
		if !ok {
			t.Fatalf("provider %s did not return a Storage: %T", provider.Name, raw)
		}

		if provider.Cleanup != nil {
			if err := provider.Cleanup(t.Context(), raw); err != nil {
				t.Errorf("error cleaning up, %v", err)
			}
		}

		ipamer := &ipamer{storage: storage}
		testName := provider.Name

		t.Run(testName, func(t *testing.T) {
			fn(t, ipamer)
		})

		if provider.PostCleanup != nil {
			if err := provider.PostCleanup(t.Context(), raw); err != nil {
				t.Errorf("error cleaning up after the test: %v", err)
			}
		}
	}
}
