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

const watchTimeFactor = 8

func NewFsWatcher(method WatchMethod, deadline time.Duration, matcher FileMatcher) (FsWatcher, error) {
	if method == WatchMethodTimer {
		interval := deadline / watchTimeFactor

		if interval > time.Minute {
			interval = time.Minute
		}

		if interval < time.Second {
			interval = time.Second
		}

		silencePeriod := deadline * watchTimeFactor

		return NewTimerFsWatcher(interval, silencePeriod, matcher)
	}

	return NewOsNotifyFsWatcher()
}
