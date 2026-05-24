package log

import (
	"bufio"
	"encoding/binary"
	"errors"
	"os"
	"sync"
)

const lenWidth = 8

var enc = binary.BigEndian

type logStore struct {
	mu     sync.RWMutex
	store  *os.File
	buf    *bufio.Writer
	size   uint64
	index  *index
	curOff uint32
}

func NewLogStore(store *os.File, index *index) *logStore {
	return &logStore{
		store: store,
		buf:   bufio.NewWriter(store),
		index: index,
	}
}

func (wal *logStore) Append(msg []byte) (off uint32, err error) {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	msgLen := len(msg)
	pos := wal.size

	err = binary.Write(wal.buf, enc, uint64(msgLen))
	if err != nil {
		return
	}

	n, err := wal.buf.Write(msg)
	if err != nil {
		return
	}

	if n != msgLen {
		err = errors.New("unable to write to wal")
		return
	}

	n += lenWidth
	wal.size += uint64(n)

	err = wal.index.Write(wal.curOff, pos)
	if err != nil {
		return
	}

	off = wal.curOff
	wal.curOff = wal.curOff + 1

	return
}

func (wal *logStore) Read(off int) ([]byte, error) {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if err := wal.buf.Flush(); err != nil {
		return nil, err
	}

	posToRead, err := wal.index.Read(uint32(off))
	if err != nil {
		return nil, err
	}

	size := make([]byte, lenWidth)

	n, err := wal.store.ReadAt(size, int64(posToRead))
	if err != nil || n < lenWidth {
		return nil, err
	}

	data := make([]byte, enc.Uint64(size))

	n, err = wal.store.ReadAt(data, int64(posToRead)+lenWidth)
	if err != nil {
		return nil, err
	}

	return data, err
}

func (s *logStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return err
	}

	return s.store.Close()
}
