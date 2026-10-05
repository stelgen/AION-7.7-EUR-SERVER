//go:build windows

package logdb

import (
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FreeMB — свободные МБ на томе каталога (GetDiskFreeSpaceExW).
func FreeMB(dir string) (int, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return 0, err
	}
	root := filepath.VolumeName(abs) + `\.`
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0, err
	}
	var freeAvail, total, freeTotal uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvail, &total, &freeTotal); err != nil {
		return 0, err
	}
	return int(freeAvail / (1024 * 1024)), nil
}

var _ = unsafe.Sizeof(0)
