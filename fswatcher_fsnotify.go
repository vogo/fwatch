package fwatch

import (
	"github.com/vogo/fsnotify"
)

type OsNotifyFsWatcher struct {
	watcher *fsnotify.Watcher
}

func (fnw *OsNotifyFsWatcher) Events() <-chan fsnotify.Event {
	return fnw.watcher.Events
}

func (fnw *OsNotifyFsWatcher) Errors() <-chan error {
	return fnw.watcher.Errors
}

func (fnw *OsNotifyFsWatcher) AddWatch(name string, flags uint32) error {
	return fnw.watcher.AddWatch(name, flags)
}

func (fnw *OsNotifyFsWatcher) Remove(name string) error {
	return fnw.watcher.Remove(name)
}

func (fnw *OsNotifyFsWatcher) Close() error {
	return fnw.watcher.Close()
}

func NewOsNotifyFsWatcher() (FsWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &OsNotifyFsWatcher{watcher: w}, nil
}
