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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vogo/logger"
)

const MaxDirFileCount = 128

var ErrTooManyDirFile = errors.New("too many files under directory")

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

	fileInfos, err := openCheckDir(dir)
	if err != nil {
		fw.handleDirError(dir, dirStat, err)

		return
	}

	var (
		filePath  string
		isDirPath bool
		pathErr   error
	)

	subDirMap := make(map[string]os.FileInfo)

	for _, fileInfo := range fileInfos {
		filePath, isDirPath, fileInfo, pathErr = unlink(filepath.Join(dir, fileInfo.Name()), fileInfo)
		if pathErr != nil {
			logger.Debugf("read file error：%v", pathErr)

			continue
		}

		if isDirPath {
			subDirMap[filePath] = fileInfo

			continue
		}

		if !dirStat.matcher(fileInfo.Name()) {
			logger.Tracef("ignore file for not match: %s", fileInfo.Name())

			continue
		}

		fw.tryAddNewFile(filePath, fileInfo, silenceDeadline)
	}

	// check sub dir
	for path, fileInfo := range subDirMap {
		fw.tryAddNewSubDir(fileInfo, path, dirStat, silenceDeadline)
	}
}

func openCheckDir(dir string) ([]os.FileInfo, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	fileInfos, err := file.Readdir(-1)
	_ = file.Close()

	if err != nil {
		return nil, err
	}

	if len(fileInfos) > MaxDirFileCount {
		return nil, fmt.Errorf("%w. dir: %s, file count: %d", ErrTooManyDirFile, dir, len(fileInfos))
	}

	return fileInfos, nil
}

func (fw *FileWatcher) handleDirError(dir string, _ *DirStat, err error) {
	logger.Debugf("ignore dir %s for %v", dir, err)

	delete(fw.dirs, dir)

	if os.IsNotExist(err) {
		return
	}

	fw.Errors <- err
}
