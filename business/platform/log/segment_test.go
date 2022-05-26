package log_test

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rsb/prolog/business/platform/log"

	data "github.com/rsb/prolog/app/api/handlers/v1"

	"github.com/stretchr/testify/require"
)

func TestSegment(t *testing.T) {
	dir, err := ioutil.TempDir("", "segment-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	want := &data.Record{Value: []byte("hello world")}

	c := log.Config{}
	c.Segment.MaxStoreBytes = 1024
	c.Segment.MaxIndexBytes = log.EntWidth * 3

	seg, err := log.NewSegment(dir, 16, c)
	require.NoError(t, err)
	require.Equal(t, uint64(16), seg.NextOffset(), seg.NextOffset())
	require.False(t, seg.IsMaxed())

	for i := uint64(0); i < 3; i++ {
		off, err := seg.Append(want)
		require.NoError(t, err)
		require.Equal(t, 16+i, off)

		got, err := seg.Read(off)
		require.NoError(t, err)
		require.Equal(t, want.Value, got.Value)
	}

	_, err = seg.Append(want)
	require.Error(t, err)

	require.True(t, errors.Is(err, io.EOF))

	// maxed index
	require.True(t, seg.IsMaxed())

	c.Segment.MaxStoreBytes = uint64(len(want.Value) * 3)
	c.Segment.MaxIndexBytes = 1024

	seg, err = log.NewSegment(dir, 16, c)
	require.NoError(t, err)

	// maxed store
	require.True(t, seg.IsMaxed())

	err = seg.Remove()
	require.NoError(t, err)

	seg, err = log.NewSegment(dir, 16, c)
	require.NoError(t, err)
	require.False(t, seg.IsMaxed())
}
