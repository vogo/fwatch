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

// start file watcher.
func (fw *FileWatcher) start() error {
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
			case <-fw.Runner.C:
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
