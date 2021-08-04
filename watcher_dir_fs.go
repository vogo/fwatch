package fwatch

import (
	"path/filepath"
	"time"

	"github.com/vogo/fsnotify"
	"github.com/vogo/logger"
)

func (fw *FileWatcher) startFsDirWatcher() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	fw.newDirWatchInit = func(dir string) {
		if dirErr := w.AddWatch(dir, FileCreateRemoveEvents); dirErr != nil {
			logger.Errorf("fs watch dir error: %v, dir: %s", dirErr, dir)
		}
	}

	go fw.fsWatchDir(w)

	return nil
}

func (fw *FileWatcher) fsWatchDir(dirWatcher *fsnotify.Watcher) {
	defer func() {
		logger.Warnf("stop watch directory")

		_ = dirWatcher.Close()
	}()

	for {
		select {
		case <-fw.Stopper.C:
			return
		case event, ok := <-dirWatcher.Events:
			if !ok {
				logger.Warnf("failed to listen watch event")

				return
			}

			fw.fsHandleDirEvent(dirWatcher, event)
		case err, ok := <-dirWatcher.Errors:
			if !ok {
				logger.Warnf("failed to listen error event")

				return
			}

			logger.Errorf("watch dir error: %v", err)
		}
	}
}

func (fw *FileWatcher) fsHandleDirEvent(dirWatcher *fsnotify.Watcher, event fsnotify.Event) {
	logger.Debugf("dir event: %v", event)

	// ignore root dir events
	if event.Name == "" || event.Name == "." {
		return
	}

	baseDir := filepath.Dir(event.Name)
	stat, ok := fw.dirs[baseDir]

	if !ok {
		logger.Warnf("unexpected event: %s", event)

		return
	}

	if IsDir(event.Name) {
		fw.fsHandleDirsEvent(dirWatcher, event, stat)

		return
	}

	fw.fsHandleFilesEvent(event, stat)
}

func (fw *FileWatcher) fsHandleDirsEvent(dirWatcher *fsnotify.Watcher, event fsnotify.Event, stat *DirStat) {
	switch event.Op {
	case fsnotify.Create:
		silenceDeadline := time.Now().Add(-fw.silenceDuration)
		fw.tryAddNewSubDir(event.Name, stat, silenceDeadline)
	case fsnotify.Remove, fsnotify.Rename:
		_ = dirWatcher.Remove(event.Name)

		delete(fw.dirs, event.Name)
	case fsnotify.Write, fsnotify.Chmod:
	}
}

func (fw *FileWatcher) fsHandleFilesEvent(event fsnotify.Event, stat *DirStat) {
	if !stat.matcher(event.Name) {
		return
	}

	switch event.Op {
	case fsnotify.Create, fsnotify.Write:
		fw.tryAddNewFile(event.Name, stat)
	case fsnotify.Remove, fsnotify.Rename:
		fw.tryRemoveFile(event.Name, stat)
	case fsnotify.Chmod:
	}
}