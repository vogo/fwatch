// +build freebsd openbsd netbsd dragonfly darwin

package fwatch

import "golang.org/x/sys/unix"

const (
	// unix.NOTE_WRIT is the event of file create/write, there is not a event only to notify file create.
	fileCreateDeleteEvents = unix.NOTE_DELETE | unix.NOTE_WRITE | unix.NOTE_RENAME

	fileWriteDeleteEvents = unix.NOTE_DELETE | unix.NOTE_WRITE | unix.NOTE_RENAME
)
