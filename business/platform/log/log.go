package log

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	data "github.com/rsb/prolog/app/api/handlers/v1"

	"github.com/rsb/failure"
)

const (
	// LenWidth number of bytes used to Store the record's length
	LenWidth             = 8
	DefaultMaxStoreBytes = 1024
	DefaultMaxIndexBytes = 1024
)

var (
	// Enc defines the encoding that we persist record sizes and index entries
	Enc = binary.BigEndian
)

type Config struct {
	Segment struct {
		MaxStoreBytes uint64
		MaxIndexBytes uint64
		InitialOffset uint64
	}
}

type Log struct {
	mu            sync.RWMutex
	Dir           string
	Config        Config
	activeSegment *Segment
	segments      []*Segment
}

func NewLog(dir string, c Config) (*Log, error) {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = DefaultMaxStoreBytes
	}

	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = DefaultMaxIndexBytes
	}

	l := Log{
		Dir:    dir,
		Config: c,
	}

	if err := l.setup(); err != nil {
		return nil, failure.Wrap(err, "l.setup failed")
	}

	return &l, nil
}

func (l *Log) setup() error {
	files, err := ioutil.ReadDir(l.Dir)
	if err != nil {
		return failure.Wrap(err, "ioutil.ReadDir failed")
	}

	var baseOffsets []uint64
	for _, file := range files {
		offStr := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))

		off, err := strconv.ParseUint(offStr, 10, 0)
		if err != nil {
			return failure.Wrap(err, "strconv.ParseUint failed")
		}
		baseOffsets = append(baseOffsets, off)
	}
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})

	for i := 0; i < len(baseOffsets); i++ {
		if err = l.newSegment(baseOffsets[i]); err != nil {
			return failure.Wrap(err, "l.newSegment failed for (%d)", baseOffsets[i])
		}
		// baseOffset contains dup for index and store so we skip the dup.
		i++
	}

	if l.segments == nil {
		if err = l.newSegment(l.Config.Segment.InitialOffset); err != nil {
			return failure.Wrap(err, "l.newSegment failed for (%d)", l.Config.Segment.InitialOffset)
		}
	}
	return nil
}

func (l *Log) Append(record *data.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	off, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, failure.Wrap(err, "l.activeSegment.Append failed")
	}

	if l.activeSegment.IsMaxed() {
		if err := l.newSegment(off + 1); err != nil {
			return 0, failure.Wrap(err, "l.newSegment failed")
		}
	}
	return off, nil
}

func (l *Log) Read(off uint64) (*data.Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var s *Segment
	for _, seg := range l.segments {
		if seg.BaseOffset() <= off && off < seg.NextOffset() {
			s = seg
			break
		}
	}

	if s == nil || s.NextOffset() <= off {
		return nil, failure.OutOfRange("invalid offset %d", off)
	}

	rec, err := s.Read(off)
	if err != nil {
		return nil, failure.Wrap(err, "s.Read failed (offset: %d)", off)
	}

	return rec, nil
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, seg := range l.segments {
		if err := seg.Close(); err != nil {
			return failure.Wrap(err, "seg.Close failed")
		}
	}

	return nil
}

func (l *Log) Remove() error {
	if err := l.Remove(); err != nil {
		return failure.Wrap(err, "l.Remove failed")
	}

	if err := os.RemoveAll(l.Dir); err != nil {
		return failure.Wrap(err, "os.RemoveAll failed")
	}

	return nil
}

func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return failure.Wrap(err, "l.Remove failed")
	}

	if err := l.setup(); err != nil {
		return failure.Wrap(err, "l.setup failed")
	}

	return nil
}

func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.segments[0].BaseOffset(), nil
}

func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	off := l.segments[len(l.segments)-1].NextOffset()
	if off == 0 {
		return 0, nil
	}

	return off - 1, nil
}

func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var segments []*Segment

	for _, s := range l.segments {
		if s.NextOffset() <= lowest+1 {
			if err := s.Remove(); err != nil {
				return failure.Wrap(err, "s.Remove failed")
			}
			continue
		}
		segments = append(segments, s)
	}
	l.segments = segments

	return nil
}

func (l *Log) Reader() io.Reader {
	l.mu.RLock()
	defer l.mu.RUnlock()

	readers := make([]io.Reader, len(l.segments))
	for i, seg := range l.segments {
		readers[i] = &originReader{seg.store, 0}
	}

	return io.MultiReader(readers...)
}

func (l *Log) newSegment(off uint64) error {
	s, err := NewSegment(l.Dir, off, l.Config)
	if err != nil {
		return failure.Wrap(err, "NewSegment failed")
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

type originReader struct {
	*Store
	off int64
}

func (o *originReader) Read(p []byte) (int, error) {
	n, err := o.ReadAt(p, o.off)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		return n, failure.Wrap(err, "o.ReadAt failed")
	}

	o.off += int64(n)
	return n, nil
}
