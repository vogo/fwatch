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
	"path/filepath"
	"time"

	"github.com/vogo/logger"
)

func (fw *FileWatcher) checkDirs(silenceDeadline time.Time) {
	for dir, stat := range fw.dirs {
		fw.checkDir(dir, stat, silenceDeadline)
	}
}

func (fw *FileWatcher) checkDir(dir string, stat *DirStat, silenceDeadline time.Time) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		fw.handleDirError(dir, stat, err)

		return
	}

	if !dirInfo.ModTime().After(stat.modTime) {
		// not need to check files in directory if dir mod time not updated.
		return
	}

	stat.modTime = dirInfo.ModTime()

	f, err := os.Open(dir)
	if err != nil {
		fw.handleDirError(dir, stat, err)

		return
	}

	fileInfos, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		fw.handleDirError(dir, stat, err)

		return
	}

	for _, info := range fileInfos {
		filePath, isDirPath, pathErr := unlink(filepath.Join(dir, info.Name()), info)
		if pathErr != nil {
			logger.Debugf("read file error：%v", pathErr)

			continue
		}

		if isDirPath {
			fw.tryAddNewSubDir(filePath, stat, silenceDeadline)

			continue
		}

		if !stat.matcher(info.Name()) || !info.ModTime().After(silenceDeadline) {
			continue
		}

		fw.tryAddNewFile(filePath, stat)
	}
}

func (fw *FileWatcher) handleDirError(dir string, _ *DirStat, err error) {
	delete(fw.dirs, dir)

	if os.IsNotExist(err) {
		return
	}

	fw.Errors <- err
}
