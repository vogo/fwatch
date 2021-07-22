package fwatch

import "golang.org/x/sys/unix"

const (
	// FileCreateDeleteEvents events for file create and delete.
	FileCreateDeleteEvents = unix.IN_CREATE |
		unix.IN_MOVE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM | unix.IN_MOVE_SELF |
		unix.IN_DELETE | unix.IN_DELETE_SELF

	// FileWriteDeleteEvents events for file write and delete.
	FileWriteDeleteEvents = unix.IN_MODIFY |
		unix.IN_MOVE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM | unix.IN_MOVE_SELF |
		unix.IN_DELETE | unix.IN_DELETE_SELF
)
