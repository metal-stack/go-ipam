package ipam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewWithStorage(t *testing.T) {
	storage := NewMemory()
	ipamer := NewWithStorage(storage)
	require.NotNil(t, ipamer)
	require.Equal(t, storage, ipamer.storage)
}

func Test_New(t *testing.T) {
	ipamer := New()
	require.NotNil(t, ipamer)
	require.NotNil(t, ipamer.storage)
}
