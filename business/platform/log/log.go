package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"

	"github.com/rsb/failure"
)

const (
	// number of bytes used to store the record's length
	lenWidth = 8
)

var (
	// enc defines the encoding that we persist record sizes and index entries
	enc = binary.BigEndian
)

type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File) (*store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, failure.ToSystem(err, "os.Stat failed")
	}
	size := uint64(fi.Size())

	s := &store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}

	return s, nil
}

func (s *store) Append(p []byte) (uint64, uint64, error) {
	var numBytes uint64
	var pos uint64
	var err error

	s.mu.Lock()
	defer s.mu.Unlock()

	pos = s.size
	if err = binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, failure.ToSystem(err, "binary.Write failed")
	}

	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, failure.ToSystem(err, "s.buf.Write failed")
	}

	w += lenWidth

	numBytes = uint64(w)
	s.size += numBytes

	return numBytes, pos, nil
}

func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return nil, failure.ToSystem(err, "s.buf.Flush failed")
	}

	size := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, failure.ToSystem(err, "s.File.ReadAt (pos: %d)", pos)
	}

	b := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, failure.ToSystem(err, "s.File.ReadAt failed (pos: %d)", pos)
	}

	return b, nil
}

func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return 0, failure.ToSystem(err, "s.buf.Flush failed (off: %d)", off)
	}

	out, err := s.File.ReadAt(p, off)
	if err != nil {
		return 0, failure.ToSystem(err, "s.File.ReadAt failed (off: %d)", off)
	}

	return out, nil
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return failure.ToSystem(err, "s.buf.Flush failed")
	}

	if err := s.File.Close(); err != nil {
		return failure.ToSystem(err, "s.File.Close failed")
	}

	return nil
}
