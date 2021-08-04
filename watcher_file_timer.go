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
