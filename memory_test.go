package ipam

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ReadPrefix(t *testing.T) {
	m := NewMemory()

	// Prefix
	p, err := m.ReadPrefix("12.0.0.0/8")
	require.NotNil(t, err)
	require.Equal(t, "prefix 12.0.0.0/8 not found", err.Error())
	require.Empty(t, p)

	prefix := Prefix{Cidr: "12.0.0.0/16"}
	p, err = m.CreatePrefix(prefix)
	require.Nil(t, err)
	require.NotNil(t, p)

	p, err = m.ReadPrefix("12.0.0.0/16")
	require.Nil(t, err)
	require.NotNil(t, p)
	require.Equal(t, "12.0.0.0/16", p.Cidr)
}

func Test_UpdatePrefix(t *testing.T) {
	m := NewMemory()

	prefix := Prefix{}
	p, err := m.UpdatePrefix(prefix)
	require.NotNil(t, err)
	require.Empty(t, p)
	require.Equal(t, "prefix not present:{  false map[] 0 map[] 0}", err.Error())

	prefix.Cidr = "1.2.3.4/24"
	p, err = m.UpdatePrefix(prefix)
	require.NotNil(t, err)
	require.Empty(t, p)
	require.Equal(t, "prefix not found:1.2.3.4/24", err.Error())
}

// ensure that locks on memory storage work
func Test_UpdatePrefix_Concurrent(t *testing.T) {
	m := NewMemory()

	for i := 0; i < 50000; i++ {

		go func(run int) {
			prefix := Prefix{}
			cidr := calcPrefix24(run) + "/24"
			prefix.Cidr = cidr

			p, err := m.CreatePrefix(prefix)
			require.Nil(t, err)
			require.NotNil(t, p)

			p, err = m.ReadPrefix(cidr)
			require.Nil(t, err)
			require.NotNil(t, p)

			p, err = m.UpdatePrefix(p)
			require.Nil(t, err)
			require.NotNil(t, p)

			p, err = m.ReadPrefix(cidr)
			require.Nil(t, err)
			require.NotNil(t, p)

			p, err = m.DeletePrefix(p)
			require.Nil(t, err)
			require.NotNil(t, p)
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
