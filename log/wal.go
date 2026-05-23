package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

const lenWidth = 8

var enc = binary.BigEndian

type WALog struct {
	mu    sync.RWMutex
	store *os.File
	buf   *bufio.Writer
	size  uint64
}

func NewLog(store *os.File) *WALog {
	return &WALog{
		store: store,
		buf:   bufio.NewWriter(store),
	}
}

func (wal *WALog) Append(msg []byte) (n int, pos uint64, err error) {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	msgLen := len(msg)
	pos = wal.size

	err = binary.Write(wal.buf, enc, uint64(msgLen))
	if err != nil {
		return 0, 0, err
	}

	n, err = wal.buf.Write(msg)
	if err != nil || n < 0 {
		return 0, 0, err
	}

	n += lenWidth
	wal.size += uint64(n)

	return
}

func (wal *WALog) Read(offset int64) ([]byte, error) {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if err := wal.buf.Flush(); err != nil {
		return nil, err
	}

	size := make([]byte, lenWidth)

	n, err := wal.store.ReadAt(size, offset)
	if err != nil || n < lenWidth {
		return nil, err
	}

	data := make([]byte, enc.Uint64(size))

	n, err = wal.store.ReadAt(data, offset+lenWidth)
	if err != nil {
		return nil, err
	}

	return data, err
}
