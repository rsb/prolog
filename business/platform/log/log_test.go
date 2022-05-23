package log_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/rsb/prolog/business/platform/log"

	"github.com/stretchr/testify/require"
)

var (
	write = []byte("Hello world")
	width = uint64(len(write)) + log.LenWidth
)

func TestStore_AppendRead(t *testing.T) {
	f, err := ioutil.TempFile("", "store_app_read_test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	s, err := log.NewStore(f)
	require.NoError(t, err)
	testAppend(t, s)
	testRead(t, s)
	testReadAt(t, s)

	s, err = log.NewStore(f)
	require.NoError(t, err)

	testRead(t, s)
}

func testAppend(t *testing.T, s *log.Store) {
	t.Helper()
	for i := uint64(1); i < 4; i++ {
		n, pos, err := s.Append(write)
		require.NoError(t, err)
		require.Equal(t, pos+n, width*i)
	}
}

func testRead(t *testing.T, s *log.Store) {
	t.Helper()

	var pos uint64
	for i := uint64(1); i < 4; i++ {
		read, err := s.Read(pos)
		require.NoError(t, err)
		require.Equal(t, write, read)
	}
}

func testReadAt(t *testing.T, s *log.Store) {
	t.Helper()

	for i, off := uint64(1), int64(0); i < 4; i++ {
		b := make([]byte, log.LenWidth)
		n, err := s.ReadAt(b, off)
		require.NoError(t, err)

		require.Equal(t, log.LenWidth, n)
		off += int64(n)

		size := log.Enc.Uint64(b)
		b = make([]byte, size)

		n, err = s.ReadAt(b, off)
		require.NoError(t, err)

		require.Equal(t, write, b)
		require.Equal(t, int(size), n)

		off += int64(n)
	}
}

func TestStore_Close(t *testing.T) {
	f, err := ioutil.TempFile("", "store_close_test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	s, err := log.NewStore(f)
	require.NoError(t, err)

	_, _, err = s.Append(write)
	require.NoError(t, err)

	f, beforeSize, err := openFile(f.Name())
	require.NoError(t, err)

	err = s.Close()
	require.NoError(t, err)

	f, afterSize, err := openFile(f.Name())
	require.NoError(t, err)

	require.True(t, afterSize > beforeSize)
}

func openFile(name string) (*os.File, int64, error) {
	f, err := os.OpenFile(
		name,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)

	if err != nil {
		return nil, 0, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, fi.Size(), nil
}
