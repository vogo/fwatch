package fwatch

import (
	"time"

	"github.com/vogo/fsnotify"
)

type FsWatcher interface {
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	AddWatch(name string, flags uint32) error
	Remove(name string) error
	Close() error
}

const half2 = 2

func NewFsWatcher(method WatchMethod, deadline time.Duration, matcher FileMatcher) (FsWatcher, error) {
	if method == WatchMethodTimer {
		interval := deadline / half2

		if interval > time.Minute {
			interval = time.Minute
		}

		if interval < time.Second {
			interval = time.Second
		}

		return NewTimerFsWatcher(interval, matcher)
	}

	return NewOsNotifyFsWatcher()
}
