package file

import (
	"os"
	"testing"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

func provide(tb testing.TB) ipam.Storage {
	fp, err := os.CreateTemp("", "go-ipam-*.json")
	if err != nil {
		tb.Fatal(err)
	}
	if err := fp.Close(); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = os.Remove(fp.Name())
	})
	return New(tb.Context(), fp.Name())
}

func TestConformance(t *testing.T) {
	suite.Run(t, provide)
}
