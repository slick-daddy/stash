//go:build darwin || freebsd
// +build darwin freebsd

package file

import (
	"io/fs"
	"syscall"
	"time"
)

// BirthTime returns the file creation/birth time from the OS.
// On macOS/FreeBSD, this comes from syscall.Stat_t.Birthtimespec.
// Truncated to seconds to match database precision.
func BirthTime(info fs.FileInfo) *time.Time {
	sys := info.Sys()
	if sys == nil {
		return nil
	}

	stat, ok := sys.(*syscall.Stat_t)
	if !ok {
		return nil
	}

	t := time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	t = t.Truncate(time.Second)
	return &t
}
