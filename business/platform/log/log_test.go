package log_test

import (
	"io/ioutil"
	"os"
	"testing"

	"google.golang.org/protobuf/proto"

	data "github.com/rsb/prolog/business/data/v1"
	"github.com/rsb/prolog/business/platform/log"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, l *log.Log){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeFailure,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
		"truncate":                          testTruncate,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "store-test")
			require.NoError(t, err)
			defer func() { _ = os.RemoveAll(dir) }()

			c := log.Config{}
			c.Segment.MaxStoreBytes = 32
			l, err := log.NewLog(dir, c)
			require.NoError(t, err)

			fn(t, l)
		})
	}
}

func testAppendRead(t *testing.T, l *log.Log) {
	ap := &data.Record{
		Value: []byte("hello world"),
	}

	off, err := l.Append(ap)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	read, err := l.Read(off)
	require.NoError(t, err)
	require.Equal(t, ap.Value, read.Value)
}

func testOutOfRangeFailure(t *testing.T, l *log.Log) {
	read, err := l.Read(1)
	require.Nil(t, read)
	require.Error(t, err)
}

func testInitExisting(t *testing.T, l *log.Log) {
	rec := &data.Record{
		Value: []byte("hello world"),
	}

	for i := 0; i < 3; i++ {
		_, err := l.Append(rec)
		require.NoError(t, err)
	}
	require.NoError(t, l.Close())

	off, err := l.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	off, err = l.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)

	n, err := log.NewLog(l.Dir, l.Config)
	require.NoError(t, err)

	off, err = n.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	off, err = n.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)
}

func testReader(t *testing.T, l *log.Log) {
	rec := &data.Record{
		Value: []byte("hello world"),
	}

	off, err := l.Append(rec)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := l.Reader()
	b, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	read := &data.Record{}
	err = proto.Unmarshal(b[log.LenWidth:], read)
	require.NoError(t, err)
	require.Equal(t, rec.Value, read.Value)
}

func testTruncate(t *testing.T, l *log.Log) {
	rec := &data.Record{
		Value: []byte("hello world"),
	}

	for i := 0; i < 3; i++ {
		_, err := l.Append(rec)
		require.NoError(t, err)
	}

	err := l.Truncate(1)
	require.NoError(t, err)

	_, err = l.Read(0)
	require.Error(t, err)
}
