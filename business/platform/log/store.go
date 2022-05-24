package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"

	"github.com/rsb/failure"
)

type Store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func NewStore(f *os.File) (*Store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, failure.ToSystem(err, "os.Stat failed")
	}
	size := uint64(fi.Size())

	s := &Store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}

	return s, nil
}

func (s *Store) Append(p []byte) (uint64, uint64, error) {
	var numBytes uint64
	var pos uint64
	var err error

	s.mu.Lock()
	defer s.mu.Unlock()

	pos = s.size
	if err = binary.Write(s.buf, Enc, uint64(len(p))); err != nil {
		return 0, 0, failure.ToSystem(err, "binary.Write failed")
	}

	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, failure.ToSystem(err, "s.buf.Write failed")
	}

	w += LenWidth

	numBytes = uint64(w)
	s.size += numBytes

	return numBytes, pos, nil
}

func (s *Store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	size := make([]byte, LenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}

	b := make([]byte, Enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+LenWidth)); err != nil {
		return b, err
	}

	return b, nil
}

func (s *Store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return 0, err
	}

	return s.File.ReadAt(p, off)
}

func (s *Store) Close() error {
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
