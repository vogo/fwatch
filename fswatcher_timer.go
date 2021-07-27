package fwatch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vogo/fsnotify"
	"github.com/vogo/logger"
)

type watchStat struct {
	watchCreate bool
	watchRename bool
	watchRemove bool
	watchWrite  bool
	modTime     time.Time
}

type TimerFsWatcher struct {
	mu   sync.Mutex
	once sync.Once
	done chan struct{}

	// a period time after which a file not being updated, then not need to check it and generate a remove event.
	silencePeriod time.Duration

	events  chan fsnotify.Event
	errors  chan error
	dirs    map[string]*watchStat
	files   map[string]*watchStat
	matcher FileMatcher
}

func NewTimerFsWatcher(interval, silencePeriod time.Duration, matcher FileMatcher) (FsWatcher, error) {
	w := &TimerFsWatcher{
		mu:            sync.Mutex{},
		once:          sync.Once{},
		done:          make(chan struct{}),
		silencePeriod: silencePeriod,
		events:        make(chan fsnotify.Event),
		errors:        make(chan error),
		dirs:          make(map[string]*watchStat, defaultMapSize),
		files:         make(map[string]*watchStat, defaultMapSize),
		matcher:       matcher,
	}

	go w.startTimerCheck(interval)

	return w, nil
}

func (tfw *TimerFsWatcher) Events() <-chan fsnotify.Event {
	return tfw.events
}

func (tfw *TimerFsWatcher) Errors() <-chan error {
	return tfw.errors
}

func (tfw *TimerFsWatcher) AddWatch(name string, flags uint32) error {
	tfw.mu.Lock()
	defer tfw.mu.Unlock()

	var (
		s  *watchStat
		ok bool
	)

	if IsDir(name) {
		s, ok = tfw.dirs[name]
		if !ok {
			s = &watchStat{modTime: time.Now()}
			tfw.dirs[name] = s
		}
	} else {
		s, ok = tfw.files[name]
		if !ok {
			s = &watchStat{modTime: time.Now()}
			tfw.files[name] = s
		}
	}

	s.watchCreate = flags&FileCreateEvents > 0
	s.watchCreate = flags&FileRenameEvents > 0
	s.watchRemove = flags&FileRemoveEvents > 0
	s.watchWrite = flags&FileWriteEvents > 0

	return nil
}

func (tfw *TimerFsWatcher) Remove(name string) error {
	tfw.mu.Lock()
	defer tfw.mu.Unlock()

	delete(tfw.files, name)
	delete(tfw.dirs, name)

	return nil
}

func (tfw *TimerFsWatcher) Close() error {
	tfw.once.Do(func() {
		close(tfw.done)
	})

	return nil
}

func (tfw *TimerFsWatcher) startTimerCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-tfw.done:
			return
		case now := <-ticker.C:
			tfw.checkFiles(now)
			tfw.checkDirs()
		}
	}
}

func (tfw *TimerFsWatcher) checkFiles(now time.Time) {
	silenceDeadline := now.Add(-tfw.silencePeriod)

	for f, stat := range tfw.files {
		info, err := os.Stat(f)
		if err != nil {
			delete(tfw.files, f)

			if os.IsNotExist(err) {
				if stat.watchRemove {
					tfw.events <- fsnotify.Event{
						Name: f,
						Op:   fsnotify.Remove,
					}
				}

				continue
			}

			tfw.errors <- err

			continue
		}

		if info == nil {
			continue
		}

		if info.ModTime().Before(silenceDeadline) {
			tfw.removeFile(f, stat)

			continue
		}

		if stat.watchWrite && info.ModTime().After(stat.modTime) {
			stat.modTime = info.ModTime()
			tfw.events <- fsnotify.Event{
				Name: f,
				Op:   fsnotify.Write,
			}
		}
	}
}

func (tfw *TimerFsWatcher) checkDirs() {
	for dir, stat := range tfw.dirs {
		dirInfo, err := os.Stat(dir)
		if err != nil {
			tfw.handleDirError(dir, stat, err)

			continue
		}

		if !dirInfo.ModTime().After(stat.modTime) {
			// not need to check files in directory if dir mod time not updated.
			continue
		}

		stat.modTime = dirInfo.ModTime()

		f, err := os.Open(dir)
		if err != nil {
			tfw.handleDirError(dir, stat, err)

			continue
		}

		fileInfos, err := f.Readdir(-1)
		_ = f.Close()

		if err != nil {
			tfw.handleDirError(dir, stat, err)

			continue
		}

		for _, info := range fileInfos {
			filePath, isDirPath, pathErr := unlink(filepath.Join(dir, info.Name()), info)
			if pathErr != nil {
				logger.Debugf("read file error：%v", pathErr)

				continue
			}

			if isDirPath {
				tfw.addDir(filePath, stat)

				continue
			}

			if !tfw.matcher(info.Name()) {
				continue
			}

			tfw.addFile(filePath, stat)
		}
	}
}

func (tfw *TimerFsWatcher) handleDirError(dir string, stat *watchStat, err error) {
	delete(tfw.dirs, dir)

	if os.IsNotExist(err) {
		if stat.watchRemove {
			tfw.events <- fsnotify.Event{
				Name: dir,
				Op:   fsnotify.Remove,
			}
		}

		return
	}

	tfw.errors <- err
}

func (tfw *TimerFsWatcher) addDir(dir string, t *watchStat) {
	if _, ok := tfw.dirs[dir]; ok {
		return
	}

	tfw.dirs[dir] = &watchStat{
		watchCreate: t.watchCreate,
		watchRename: t.watchRename,
		watchRemove: t.watchRemove,
		watchWrite:  t.watchWrite,
		modTime:     time.Now(),
	}
}

func (tfw *TimerFsWatcher) addFile(path string, t *watchStat) {
	if _, ok := tfw.files[path]; ok {
		return
	}

	tfw.files[path] = &watchStat{
		watchCreate: t.watchCreate,
		watchRename: t.watchRename,
		watchRemove: t.watchRemove,
		watchWrite:  t.watchWrite,
		modTime:     time.Now(),
	}

	tfw.events <- fsnotify.Event{
		Name: path,
		Op:   fsnotify.Create,
	}
}

func (tfw *TimerFsWatcher) removeFile(f string, stat *watchStat) {
	delete(tfw.files, f)

	if stat.watchRemove {
		tfw.events <- fsnotify.Event{
			Name: f,
			Op:   fsnotify.Remove,
		}
	}
}
