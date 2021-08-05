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

import (
	"os"
	"time"
)

func (fw *FileWatcher) checkFiles(inactiveDeadline, silenceDeadline time.Time) {
	for f, stat := range fw.files {
		fw.checkFile(f, stat, inactiveDeadline, silenceDeadline)
	}
}

func (fw *FileWatcher) checkFile(f string, stat *FileStat, inactiveDeadline, silenceDeadline time.Time) {
	info, err := os.Stat(f)
	if err != nil {
		delete(fw.files, f)

		if os.IsNotExist(err) {
			fw.Events <- &WatchEvent{
				Name:  f,
				Event: Remove,
			}

			return
		}

		fw.Errors <- err

		return
	}

	if info == nil {
		return
	}

	if stat.active {
		if info.ModTime().Before(inactiveDeadline) {
			stat.active = false
			fw.Events <- &WatchEvent{
				Name:  f,
				Event: Inactive,
			}
		}
	} else {
		if info.ModTime().After(stat.modTime) {
			stat.active = true
			fw.Events <- &WatchEvent{
				Name:  f,
				Event: Write,
			}
		} else if info.ModTime().Before(silenceDeadline) {
			fw.removeFile(f, stat)

			return
		}
	}

	stat.modTime = info.ModTime()
}

func (fw *FileWatcher) removeFile(f string, _ *FileStat) {
	delete(fw.files, f)

	fw.Events <- &WatchEvent{
		Name:  f,
		Event: Remove,
	}
}
