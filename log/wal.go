package log

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"sync"
)

const (
	lenWidth      = 8
	checksumWidth = 4
)

// BigEndian is standard for network oriented applications
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

	pos := wal.size

	msgLen := uint64(len(msg))
	lenBuf := make([]byte, lenWidth)
	enc.PutUint64(lenBuf, msgLen)

	crc := crc32.NewIEEE()
	crc.Write(lenBuf)
	crc.Write(msg)
	checksum := crc.Sum32()
	checksumBuf := make([]byte, checksumWidth)
	binary.BigEndian.PutUint32(checksumBuf, checksum)

	if _, err = wal.buf.Write(lenBuf); err != nil {
		return
	}

	if _, err = wal.buf.Write(checksumBuf); err != nil {
		return
	}

	n, err := wal.buf.Write(msg)
	if err != nil {
		return
	}

	if uint64(n) != msgLen {
		err = errors.New("unable to write to wal")
		return
	}

	totalBytesWritten := lenWidth + checksumWidth + len(msg)
	wal.size += uint64(totalBytesWritten)

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

	lenBuf := make([]byte, lenWidth)

	_, err = wal.store.ReadAt(lenBuf, int64(posToRead))
	if err != nil {
		return nil, err
	}

	checksumBuf := make([]byte, checksumWidth)

	_, err = wal.store.ReadAt(checksumBuf, lenWidth+int64(posToRead))
	if err != nil {
		return nil, err
	}

	expectedChecksum := enc.Uint32(checksumBuf)

	// FIX: If lenbuf is curroupted below line can allocate huge amount of memory. Add some cap later.
	dataLen := enc.Uint64(lenBuf)
	data := make([]byte, dataLen)

	_, err = wal.store.ReadAt(data, int64(posToRead)+lenWidth+checksumWidth)
	if err != nil {
		return nil, err
	}

	crc := crc32.NewIEEE()
	crc.Write(lenBuf)
	crc.Write(data)

	actualChecksum := crc.Sum32()

	if actualChecksum != expectedChecksum {
		return nil, errors.New("corrupted WAL entry: checksum mismatch")
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
