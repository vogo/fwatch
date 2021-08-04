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
