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

	// FileRemoveEvents events for file remove.
	FileRemoveEvents = unix.IN_DELETE | unix.IN_DELETE_SELF

	// FileCreateRemoveEvents events for file create and remove.
	FileCreateRemoveEvents = FileCreateEvents | FileRemoveEvents | FileRenameEvents

	// FileWriteRemoveEvents events for file write and remove.
	FileWriteRemoveEvents = FileWriteEvents | FileRemoveEvents | FileRenameEvents
)
