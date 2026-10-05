package etcd

import (
	"context"
	"testing"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

func provide(tb testing.TB) ipam.Storage {
	_, s, err := startEtcdForTest(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	ctx := tb.Context()
	if err := suite.Cleanup(ctx, s); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = suite.Cleanup(context.Background(), s)
	})
	return s
}

func TestConformance(t *testing.T) {
	suite.Run(t, provide)
}
