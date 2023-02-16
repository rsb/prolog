package log

import (
	"fmt"
	"os"
	"path"

	data "github.com/rsb/prolog/app/api/handlers/v1"
	"google.golang.org/protobuf/proto"

	"github.com/rsb/failure"
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

func (s *Segment) NextOffset() uint64 {
	return s.nextOffset
}

func (s *Segment) BaseOffset() uint64 {
	return s.baseOffset
}

// Append writes the record to the segment and returns the newly appended
// record's offset. The segment appends the record in a two-step process:
// it appends the data to the store and then adds an index entry. Since the
// offsets are relative to the base offset, we subtract the segment's next
// offset from its base offset (which are both absolute offsets) to get the
// entry's relative offset in the segment. We then increment the next offset
// to prep for a future append.
func (s *Segment) Append(record *data.Record) (uint64, error) {
	var err error

	cur := s.nextOffset
	record.Offset = cur

	p, err := proto.Marshal(record)
	if err != nil {
		return 0, failure.ToSystem(err, "proto.Marshal failed")
	}

	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, failure.Wrap(err, "s.store.Append failed")
	}

	if err = s.index.Write(uint32(s.nextOffset-s.baseOffset), pos); err != nil {
		return 0, failure.Wrap(err, "s.index.Write failed")
	}

	s.nextOffset++
	return cur, nil
}

// Read fetches the record for a given offset. Similar to writes, to read a
// record the segment must first translate the absolute index into a relative
// offset and get the associated index entry. Once it has the index try, the
// segment can go straight to the record's position in the store and read the
// proper amount of data.
func (s *Segment) Read(off uint64) (*data.Record, error) {

	in := int64(off - s.baseOffset)
	_, pos, err := s.index.Read(in)
	if err != nil {
		return nil, failure.Wrap(err, "s.index.Read failed (%d)", in)
	}

	p, err := s.store.Read(pos)
	if err != nil {
		return nil, failure.Wrap(err, "s.store.Read failed (%d)", pos)
	}

	record := data.Record{}
	if err = proto.Unmarshal(p, &record); err != nil {
		return nil, failure.ToSystem(err, "proto.Unmarshal failed")
	}

	return &record, nil
}

// IsMaxed returns whether the segment has reached its max size, either by
// writing too much to the store or index. If you wrote a small number of long
// logs, then you'd hit the segment bytes limit; if you wrote a lot of small
// logs, then you'd hit the index bytes limit. The log uses this method to know
// it needs to create a new segment.
func (s *Segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes ||
		s.index.size >= s.config.Segment.MaxIndexBytes
}

// Remove closes the segment and removes the index and store files
func (s *Segment) Remove() error {
	if err := s.Close(); err != nil {
		return failure.Wrap(err, "s.Close failed")
	}

	if err := os.Remove(s.index.Name()); err != nil {
		return failure.ToSystem(err, "os.Remove failed for index")
	}

	if err := os.Remove(s.store.Name()); err != nil {
		return failure.ToSystem(err, "os.Remove failed for store")
	}

	return nil
}

// Close will close both the index and the store files
func (s *Segment) Close() error {
	if err := s.index.Close(); err != nil {
		return failure.Wrap(err, "s.index.Close failed")
	}

	if err := s.store.Close(); err != nil {
		return failure.Wrap(err, "s.store.Close failed")
	}

	return nil
}

// NearestMultiple returns the nearest and lesser multiple of k in j.
// For example: (9, 4) == 8. We take the lesser multiple to make sure we
// stay under the user's disk capacity.
func NearestMultiple(j, k uint64) uint64 {
	if j >= 0 {
		return (j / k) * k
	}

	return ((j - k + 1) / k) * k
}
