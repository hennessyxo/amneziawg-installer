//go:build !linux

package sysstat

// readDisk is a no-op off Linux (the panel runs on Linux; this keeps dev builds
// on other platforms compiling).
func readDisk(path string) (used, total uint64) { return 0, 0 }
