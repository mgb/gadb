package gadb

import (
	"os"
	"testing"
)

func TestFileInfo_osFileInfo(_ *testing.T) {
	_ = os.FileInfo(fileInfo{})
}
