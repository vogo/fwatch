package fwatch

import "golang.org/x/sys/unix"

const (

	// FileCreateEvents events for file create, unix.NOTE_WRIT is the event of file create/write,
	// there is not a event only to notify file create.
	FileCreateEvents = unix.IN_CREATE

	// FileWriteEvents events for file write.
	FileWriteEvents = unix.IN_MODIFY

	// FileRenameEvents events for file rename.
	FileRenameEvents = unix.IN_MOVE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM | unix.IN_MOVE_SELF

	// FileDeleteEvents events for file delete.
	FileDeleteEvents = unix.IN_DELETE | unix.IN_DELETE_SELF

	// FileCreateDeleteEvents events for file create and delete.
	FileCreateDeleteEvents = FileCreateEvents | FileDeleteEvents | FileRenameEvents

	// FileWriteDeleteEvents events for file write and delete.
	FileWriteDeleteEvents = FileWriteEvents | FileDeleteEvents | FileRenameEvents
)
