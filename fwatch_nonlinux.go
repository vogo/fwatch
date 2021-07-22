// +build freebsd openbsd netbsd dragonfly darwin

package fwatch

import "golang.org/x/sys/unix"

const (
	// FileCreateEvents events for file create, unix.NOTE_WRIT is the event of file create/write,
	// there is not a event only to notify file create.
	FileCreateEvents = unix.NOTE_WRITE

	// FileWriteEvents events for file write.
	FileWriteEvents = unix.NOTE_WRITE

	// FileRenameEvents events for file rename.
	FileRenameEvents = unix.NOTE_RENAME

	// FileDeleteEvents events for file delete.
	FileDeleteEvents = unix.NOTE_DELETE

	// FileCreateDeleteEvents events for file create and delete.
	FileCreateDeleteEvents = FileCreateEvents | FileDeleteEvents | FileRenameEvents

	// FileWriteDeleteEvents events for file write and delete.
	FileWriteDeleteEvents = FileWriteEvents | FileDeleteEvents | FileRenameEvents
)
