package log

import (
	"fmt"
	"os"
	"path"

	"github.com/rsb/failure"
	// "google.golang.org/protobuf/proto"
)

type Segment struct {
	store      *Store
	index      *Index
	baseOffset uint64
	nextOffset uint64
	config     Config
}

func NewSegment(dir string, baseOffset uint64, c Config) (*Segment, error) {
	var err error
	var nextOffset uint64

	sf := path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store"))
	storeFile, err := os.OpenFile(sf, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, failure.ToSystem(err, "os.OpenFile failed for (storeFile)")
	}

	store, err := NewStore(storeFile)
	if err != nil {
		return nil, failure.Wrap(err, "NewSTore failed")
	}

	idxF := path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index"))
	idxFile, err := os.OpenFile(idxF, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, failure.ToSystem(err, "os.OpenFile failed for (indexFile)")
	}

	idx, err := NewIndex(idxFile, c)
	if err != nil {
		return nil, failure.Wrap(err, "NewIndex failed")
	}

	off, _, err := idx.Read(-1)
	if err != nil {
		nextOffset = baseOffset
	} else {
		nextOffset = baseOffset + uint64(off) + 1
	}

	s := Segment{
		store:      store,
		index:      idx,
		baseOffset: baseOffset,
		nextOffset: nextOffset,
		config:     c,
	}

	return &s, nil
}

// func (s *Segment) Append(record *business.Record) (uint64, error) {
// 	var offset uint64
// 	var err error
//
// 	p, err := proto.Marshal(record)
// 	if err != nil {
// 		return 0, failure.ToSystem(err, "proto.Marshal failed")
// 	}
// }
