package postgres

import (
	"context"
	"testing"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

func cleanProvider(start func(ctx context.Context) (ipam.Storage, error)) func(testing.TB) ipam.Storage {
	return func(tb testing.TB) ipam.Storage {
		s, err := start(tb.Context())
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
}

func startPG(ctx context.Context) (ipam.Storage, error) {
	_, s, err := startPostgres(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func startCR(ctx context.Context) (ipam.Storage, error) {
	_, s, err := startCockroach(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func TestConformance(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		suite.Run(t, cleanProvider(startPG))
	})
	t.Run("Cockroach", func(t *testing.T) {
		suite.Run(t, cleanProvider(startCR))
	})
}
