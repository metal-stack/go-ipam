package ipam

import (
	"fmt"
	"testing"

	"context"

	"github.com/stretchr/testify/require"
)

func Test_ReadPrefix(t *testing.T) {
	ctx := context.Background()
	m := NewMemory(ctx)

	// Prefix
	p, err := m.ReadPrefix(ctx, "12.0.0.0/8", defaultNamespace)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
	require.Equal(t, "NotFound prefix 12.0.0.0/8 not found", err.Error())
	require.Empty(t, p)

	prefix := Prefix{Cidr: "12.0.0.0/16"}
	p, err = m.CreatePrefix(ctx, prefix, defaultNamespace)
	require.NoError(t, err)
	require.NotNil(t, p)

	p, err = m.ReadPrefix(ctx, "12.0.0.0/16", defaultNamespace)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, "12.0.0.0/16", p.Cidr)
}

func Test_UpdatePrefix(t *testing.T) {
	ctx := context.Background()
	m := NewMemory(ctx)

	prefix := Prefix{}
	p, err := m.UpdatePrefix(ctx, prefix, defaultNamespace)
	require.Error(t, err)
	require.Empty(t, p)
	require.Equal(t, "prefix not present:{  false map[] 0 map[] 1}", err.Error())

	prefix.Cidr = "1.2.3.4/24"
	p, err = m.UpdatePrefix(ctx, prefix, defaultNamespace)
	require.Error(t, err)
	require.Empty(t, p)
	require.Equal(t, "prefix not found:1.2.3.4/24", err.Error())
}

// ensure that locks on memory storage work
func Test_UpdatePrefix_Concurrent(t *testing.T) {
	ctx := context.Background()
	m := NewMemory(ctx)

	for i := range 50000 {

		go func(run int) {
			prefix := Prefix{}
			cidr := calcPrefix24(run) + "/24"
			prefix.Cidr = cidr

			_, err := m.CreatePrefix(ctx, prefix, defaultNamespace)
			if err != nil {
				t.Error(t, err)
			}

			p, err := m.ReadPrefix(ctx, cidr, defaultNamespace)
			if err != nil {
				t.Error(t, err)
			}

			_, err = m.UpdatePrefix(ctx, p, defaultNamespace)
			if err != nil {
				t.Error(t, err)
			}

			p, err = m.ReadPrefix(ctx, cidr, defaultNamespace)
			if err != nil {
				t.Error(t, err)
			}

			_, err = m.DeletePrefix(ctx, p, defaultNamespace)
			if err != nil {
				t.Error(t, err)
			}
		}(i)
	}
}

// calcs distinct /24 prefix for given test run
func calcPrefix24(run int) string {
	i3 := run % 256
	i2 := (run / 256) % 256
	i1 := (run / 65536) % 256

	return fmt.Sprintf("%d.%d.%d.0", i1, i2, i3)
}
