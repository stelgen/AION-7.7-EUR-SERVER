//go:build !windows

package logdb

import (
	"golang.org/x/sys/unix"
)

// FreeMB — свободные МБ на ФС каталога (linux-сборка).
func FreeMB(dir string) (int, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int(uint64(st.Bavail) * uint64(st.Bsize) / (1024 * 1024)), nil
}
