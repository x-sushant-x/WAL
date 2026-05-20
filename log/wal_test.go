package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWAL(t *testing.T) {
	f, err := os.CreateTemp(".", "test.bin")
	defer os.Remove(f.Name())

	require.NoError(t, err)

	log := NewLog(f)

	n, offset, err := log.Append([]byte("Hello"))
	require.NoError(t, err)

	require.Equal(t, 8+5, n)
	require.Equal(t, uint64(0), offset)

	data, err := log.Read(int64(offset))
	require.NoError(t, err)

	require.Equal(t, data, []byte("Hello"))
}
