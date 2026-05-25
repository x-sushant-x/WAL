package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWAL(t *testing.T) {
	logFile, err := os.CreateTemp(".", "log.bin")
	require.NoError(t, err)
	// defer os.Remove(logFile.Name())

	indexFile, err := os.CreateTemp(".", "index.bin")
	require.NoError(t, err)
	// defer os.Remove(indexFile.Name())

	index := NewIndex(indexFile)

	log := NewLogStore(logFile, index)

	off, err := log.Append([]byte("Hello"))
	require.NoError(t, err)
	require.Equal(t, off, uint32(0))

	off, err = log.Append([]byte("World"))
	require.NoError(t, err)
	require.Equal(t, off, uint32(1))

	data, err := log.Read(0)
	require.NoError(t, err)
	require.Equal(t, []byte("Hello"), data)

	data, err = log.Read(1)
	require.NoError(t, err)
	require.Equal(t, []byte("World"), data)
}
