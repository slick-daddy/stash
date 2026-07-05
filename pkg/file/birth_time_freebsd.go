//go:build freebsd
// +build freebsd

package file

import (
	"io/fs"
	"time"

	"golang.org/x/sys/unix"
)

// BirthTime returns the file creation/birth time from the OS.
// On FreeBSD, this comes from unix.Stat_t.Btim.
// Uses golang.org/x/sys/unix for stable field names across Go versions.
// Truncated to seconds to match database precision.
func BirthTime(info fs.FileInfo) *time.Time {
	sys := info.Sys()
	if sys == nil {
		return nil
	}

	stat, ok := sys.(*unix.Stat_t)
	if !ok {
		return nil
	}

	t := time.Unix(stat.Btim.Sec, stat.Btim.Nsec)
	t = t.Truncate(time.Second)
	return &t
}
