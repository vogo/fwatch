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

func (fw *FileWatcher) checkDir(dir string, dirStat *DirStat, silenceDeadline time.Time) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		fw.handleDirError(dir, dirStat, err)

		return
	}

	fw.checkDirInfo(dir, dirInfo, dirStat, silenceDeadline)
}

func (fw *FileWatcher) checkDirInfo(dir string, dirInfo os.FileInfo, dirStat *DirStat, silenceDeadline time.Time) {
	// dir mod time is updated only when creating or removing sub files.
	// not need to check files in directory if dir mod time not updated.
	if !dirInfo.ModTime().After(dirStat.modTime) {
		logger.Tracef("ignore not updated dir: %s", dirInfo.Name())

		return
	}

	dirStat.modTime = dirInfo.ModTime()

	logger.Debugf("start check dir: %s", dir)

	f, err := os.Open(dir)
	if err != nil {
		fw.handleDirError(dir, dirStat, err)

		return
	}

	fileInfos, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		fw.handleDirError(dir, dirStat, err)

		return
	}

	var (
		filePath  string
		isDirPath bool
		pathErr   error
	)

	for _, fileInfo := range fileInfos {
		filePath, isDirPath, fileInfo, pathErr = unlink(filepath.Join(dir, fileInfo.Name()), fileInfo)
		if pathErr != nil {
			logger.Debugf("read file error：%v", pathErr)

			continue
		}

		if isDirPath {
			fw.tryAddNewSubDir(fileInfo, filePath, dirStat, silenceDeadline)

			continue
		}

		if !dirStat.matcher(fileInfo.Name()) || !fileInfo.ModTime().After(silenceDeadline) {
			continue
		}

		fw.tryAddNewFile(filePath, fileInfo, silenceDeadline)
	}
}

func (fw *FileWatcher) handleDirError(dir string, _ *DirStat, err error) {
	delete(fw.dirs, dir)

	if os.IsNotExist(err) {
		return
	}

	fw.Errors <- err
}
