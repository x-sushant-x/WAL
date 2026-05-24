// index is NOT thread-safe.
// All access must be synchronized by the caller (logStore).

package log

import (
	"bufio"
	"encoding/binary"
	"errors"
	"os"
)

const (
	offWidth = 4
	posWidth = 8
	totWidth = offWidth + posWidth
)

var indexEnc = binary.BigEndian

type index struct {
	f    *os.File
	buf  *bufio.Writer
	size uint64
}

func NewIndex(f *os.File) *index {
	return &index{
		f:   f,
		buf: bufio.NewWriter(f),
	}
}

func (i *index) Name() string {
	return i.f.Name()
}

func (i *index) Close() error {
	if err := i.buf.Flush(); err != nil {
		return err
	}

	if err := i.f.Close(); err != nil {
		return err
	}

	return nil
}

func (i *index) Write(off uint32, pos uint64) error {
	err := binary.Write(i.buf, indexEnc, uint32(off))
	if err != nil {
		return err
	}

	err = binary.Write(i.buf, indexEnc, uint64(pos))
	if err != nil {
		return err
	}

	i.size += totWidth

	return err
}

func (i *index) Read(off uint32) (pos uint64, err error) {
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
