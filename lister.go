package main

import (
	"io"
	"os"
)

// ListerAt implements sftp.ListerAt
type ListerAt []os.FileInfo

func (l ListerAt) ListAt(f []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(f, l[offset:])
	if n < len(f) {
		return n, io.EOF
	}

	return n, nil
}
