package log

import (
	"bufio"
	"encoding/binary"
	"errors"
	"os"
	"sync"
)

const (
	offWidth = 4
	posWidth = 8
	totWidth = offWidth + posWidth
)

var indexEnc = binary.BigEndian

type Index struct {
	f   *os.File
	mu  sync.RWMutex
	buf *bufio.Writer
}

func NewIndex(f *os.File) *Index {
	return &Index{
		f:   f,
		buf: bufio.NewWriter(f),
	}
}

func (i *Index) Write(off uint32, pos uint64) error {
	err := binary.Write(i.buf, indexEnc, uint32(off))
	if err != nil {
		return err
	}

	err = binary.Write(i.buf, indexEnc, uint64(pos))
	if err != nil {
		return err
	}

	return err
}

func (i *Index) Read(off uint32) (pos uint64, err error) {
	if err := i.buf.Flush(); err != nil {
		return 0, err
	}

	posInIndex := off * totWidth
	posInIndex += offWidth

	data := make([]byte, 8)

	n, err := i.f.ReadAt(data, int64(posInIndex))
	if err != nil {
		return 0, err
	}

	if n < 8 {
		return 0, errors.New("unable to read position bytes")
	}

	pos = indexEnc.Uint64(data)
	return pos, err
}
