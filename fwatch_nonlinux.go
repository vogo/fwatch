//go:build freebsd || openbsd || netbsd || dragonfly || darwin
// +build freebsd openbsd netbsd dragonfly darwin

/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
