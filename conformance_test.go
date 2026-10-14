package ipam_test

import (
	"testing"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/test/backends"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

func TestConformance(t *testing.T) {
	suite.Run(t, func(tb testing.TB) ipam.Storage {
		return ipam.NewMemory(tb.Context())
	})
}

// BenchmarkConformance benchmarks every supported backend so their results can
// be compared in a single run.
func BenchmarkConformance(b *testing.B) {
	suite.RunBenchmarks(b, backends.Providers())
}
