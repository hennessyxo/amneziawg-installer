//go:build linux

package sysstat

import "syscall"

// readDisk returns used and total bytes for the filesystem holding path.
func readDisk(path string) (used, total uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	bs := uint64(st.Bsize)
	total = st.Blocks * bs
	free := st.Bavail * bs
	if total > free {
		used = total - free
	}
	return used, total
}
