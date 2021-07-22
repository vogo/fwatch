// +build freebsd openbsd netbsd dragonfly darwin

package fwatch

import "golang.org/x/sys/unix"

const (
	// FileCreateDeleteEvents unix.NOTE_WRIT is the event of file create/write, there is not a event only to notify file create.
	FileCreateDeleteEvents = unix.NOTE_DELETE | unix.NOTE_WRITE | unix.NOTE_RENAME

	// FileWriteDeleteEvents events for file write and delete.
	FileWriteDeleteEvents = unix.NOTE_DELETE | unix.NOTE_WRITE | unix.NOTE_RENAME
)
