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

	// FileRemoveEvents events for file remove.
	FileRemoveEvents = unix.NOTE_DELETE

	// FileCreateRemoveEvents events for file create and remove.
	FileCreateRemoveEvents = FileCreateEvents | FileRemoveEvents | FileRenameEvents

	// FileWriteRemoveEvents events for file write and remove.
	FileWriteRemoveEvents = FileWriteEvents | FileRemoveEvents | FileRenameEvents
)
