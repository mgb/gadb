package gadb

import (
	"os"
	"time"
)

const (
	dirBit = 1 << 14
)

type fileInfo struct {
	name    string
	mode    os.FileMode
	size    uint32
	modTime time.Time
}

func (f fileInfo) Name() string {
	return f.name
}

func (f fileInfo) Size() int64 {
	return int64(f.size)
}

func (f fileInfo) Mode() os.FileMode {
	return f.mode
}

func (f fileInfo) ModTime() time.Time {
	return f.modTime
}

func (f fileInfo) IsDir() bool {
	return f.mode&dirBit != 0
}

func (f fileInfo) Sys() interface{} {
	return nil
}
