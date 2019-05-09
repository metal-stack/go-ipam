package ipam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_UpdatePrefix(t *testing.T) {
	m := NewMemory()

	prefix := &Prefix{}
	p, err := m.UpdatePrefix(prefix)
	require.NotNil(t, err)
	require.Nil(t, p)
	require.Equal(t, "prefix not present:", err.Error())

	prefix.Cidr = "1.2.3.4/24"
	p, err = m.UpdatePrefix(prefix)
	require.NotNil(t, err)
	require.Nil(t, p)
	require.Equal(t, "prefix not found:1.2.3.4/24", err.Error())
}
