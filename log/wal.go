/*
 * This code is responsible for writing data to log files in following format:
 * [length][checksum][data]
 *
 * TODO:
 * 1. Handle Partial Reads & Writes.
 * 2. Handle error roleback.
 * 3. If lenbuf is curroupted huge amount of memory can be allocated. Add some cap.
 */

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

/*
 * This function perform following steps:
 * 1. Calculate message length and convert it into 8 byte big endian format.
 * 2. Generate and store a checksum from the combination of msgLen + msg.
 * 3. Store data into log file in following format: [length][checksum][data]
 */
func (wal *logStore) Append(msg []byte) (off uint32, err error) {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	/*
	 * pos tells the position at which current entry is being appended in log file.
	 * This is later sent to index module which maintain the indexing of each entry for optimized lookup while reading.
	 */
	pos := wal.size

	msgLen := uint64(len(msg))
	lenBuf := make([]byte, lenWidth)
	enc.PutUint64(lenBuf, msgLen)

	/*
	 * We are using CRC32 for checksum because:
	 * 1. It is exactly designed for detecting corruption in streaming and storage systems.
	 * 2. It is extremely fst.
	 * 3. It take only 4 byte of space.
	 */
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

/*
 * This function perform following steps:
 * 1. Flush data to make sure everything is written in disk before reading. Sometime data is buffered but not written in disk.
 * 2. Ask index module for the position where data is stored in log file for a particular message offset.
 * 3. Read the checksum.
 * 4. Read the message.
 * 5. Generate a checksum with length + message.
 * 6. Compare if generated checksum and stored checksum is equal or not.
 */
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
