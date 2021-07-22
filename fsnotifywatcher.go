package fwatch

import (
	"time"

	"github.com/vogo/fsnotify"
)

type FsNotifyWatcher interface {
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	AddWatch(name string, flags uint32) error
	Remove(name string) error
	Close() error
}

type OsEventFsNotifyWatcher struct {
	watcher *fsnotify.Watcher
}

func (osw *OsEventFsNotifyWatcher) Events() <-chan fsnotify.Event {
	return osw.watcher.Events
}

func (osw *OsEventFsNotifyWatcher) Errors() <-chan error {
	return osw.watcher.Errors
}

func (osw *OsEventFsNotifyWatcher) AddWatch(name string, flags uint32) error {
	return osw.watcher.AddWatch(name, flags)
}

func (osw *OsEventFsNotifyWatcher) Remove(name string) error {
	return osw.watcher.Remove(name)
}

func (osw *OsEventFsNotifyWatcher) Close() error {
	return osw.watcher.Close()
}

func NewOsEventFsNotifyWatcher() (FsNotifyWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &OsEventFsNotifyWatcher{watcher: w}, nil
}

const half2 = 2

func NewFsNotifyWatcher(method WatchMethod, deadline time.Duration, matcher FileMatcher) (FsNotifyWatcher, error) {
	if method == WatchMethodTimer {
		interval := deadline / half2

		if interval > time.Minute {
			interval = time.Minute
		}

		if interval < time.Second {
			interval = time.Second
		}

		return NewTimerCheckFsNotifyWatcher(interval, matcher)
	}

	return NewOsEventFsNotifyWatcher()
}
