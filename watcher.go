package fwatch

import (
	"time"
)

const (
	watchTimeFactor           = 3
	maxFsWatcherTimerInterval = time.Minute
	minFsWatcherTimerInterval = time.Second
)

func calcInterval(deadline time.Duration) time.Duration {
	interval := deadline / watchTimeFactor

	if interval > maxFsWatcherTimerInterval {
		interval = maxFsWatcherTimerInterval
	}

	if interval < minFsWatcherTimerInterval {
		interval = minFsWatcherTimerInterval
	}

	return interval
}

func (fw *FileWatcher) Start() error {
	ticker := time.NewTicker(calcInterval(fw.inactiveDuration))

	if fw.method == WatchMethodFS {
		if err := fw.startFsDirWatcher(); err != nil {
			return err
		}
	}

	// start ticker.
	go func() {
		for {
			select {
			case <-fw.Stopper.C:
				return
			case now := <-ticker.C:
				fw.timerCheck(now)
			}
		}
	}()

	return nil
}

func (fw *FileWatcher) timerCheck(now time.Time) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	inactiveDeadline := now.Add(-fw.inactiveDuration)
	silenceDeadline := now.Add(-fw.silenceDuration)

	// check files.
	fw.checkFiles(inactiveDeadline, silenceDeadline)

	// check dirs.
	fw.timerDirsChecker(silenceDeadline)

	// move new dirs to watch dirs map.
	for dir, stat := range fw.newDirs {
		fw.dirs[dir] = stat

		delete(fw.newDirs, dir)

		fw.newDirWatchInit(dir)
	}

	// move new files to watch files map.
	for f, stat := range fw.newFiles {
		fw.files[f] = stat

		delete(fw.newFiles, f)
	}
}
