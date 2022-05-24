package log

import (
	"io"
	"os"

	"github.com/rsb/failure"

	"github.com/tysonmote/gommap"
)

var (
	OffWidth uint64 = 4
	PosWidth uint64 = 8
	EntWidth        = OffWidth + PosWidth
)

type Config struct {
	Segment struct {
		MaxStoreBytes uint64
		MaxIndexBytes uint64
		InitialOffset uint64
	}
}

type Index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

func NewIndex(f *os.File, c Config) (*Index, error) {
	if f == nil {
		return nil, failure.System("f file is nil")
	}
	idx := Index{file: f}
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, failure.ToSystem(err, "os.Stat failed")
	}

	idx.size = uint64(fi.Size())
	if err = os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, failure.ToSystem(err, "os.Truncate failed")
	}

	idx.mmap, err = gommap.Map(idx.file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE, gommap.MAP_SHARED)
	if err != nil {
		return nil, failure.ToSystem(err, "gommap.Map failed")
	}

	return &idx, nil
}

func (i *Index) Close() error {
	var err error
	if err = i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return failure.ToSystem(err, "i.mmap.Sync failed")
	}

	if err = i.file.Sync(); err != nil {
		return failure.ToSystem(err, "i.file.Sync failed")
	}

	if err = i.file.Truncate(int64(i.size)); err != nil {
		return failure.ToSystem(err, "i.file.Truncate failed")
	}

	if err = i.file.Close(); err != nil {
		return failure.ToSystem(err, "i.file.Close failed")
	}

	return nil
}

func (i *Index) Read(in int64) (uint32, uint64, error) {
	var out uint32
	var pos uint64

	if i.size == 0 {
		return out, pos, io.EOF
	}

	out = uint32(in)
	if in == -1 {
		out = uint32((i.size / EntWidth) - 1)
	}

	pos = uint64(out) * EntWidth
	if i.size < pos+EntWidth {
		return 0, 0, io.EOF
	}

	out = Enc.Uint32(i.mmap[pos : pos+OffWidth])
	pos = Enc.Uint64(i.mmap[pos+OffWidth : pos+EntWidth])

	return out, pos, nil
}

func (i *Index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+EntWidth {
		return io.EOF
	}

	Enc.PutUint32(i.mmap[i.size:i.size+OffWidth], off)
	Enc.PutUint64(i.mmap[i.size+OffWidth:i.size+EntWidth], pos)
	i.size += uint64(EntWidth)
	return nil
}

func (i *Index) Name() string {
	return i.file.Name()
}
