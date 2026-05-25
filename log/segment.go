package log

import (
	"fmt"
	"os"
	"path"
)

type segment struct {
	store            *logStore
	index            *index
	baseOff, nextOff uint64
}

func newSegment(baseOff uint64, dir string) (*segment, error) {
	s := segment{
		baseOff: baseOff,
	}

	storeFileName := path.Join(dir, fmt.Sprintf("%d.store", baseOff))
	storeFile, err := os.OpenFile(storeFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	store, err := newLogStore(storeFile)
	if err != nil {
		return nil, err
	}

	indexFileName := path.Join(dir, fmt.Sprintf("%d%s", baseOff, ".index"))
	indexFile, err := os.OpenFile(indexFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	index, err := newIndex(indexFile)
	if err != nil {
		return nil, err
	}

	s.index = index
	s.store = store

	// TODO - A lot of work pending here.
}
