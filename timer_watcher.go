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
	watchDelete bool
	watchWrite  bool
	modTime     time.Time
}

type TimerCheckFsNotifyWatcher struct {
	mu      sync.Mutex
	once    sync.Once
	done    chan struct{}
	events  chan fsnotify.Event
	errors  chan error
	dirs    map[string]*watchStat
	files   map[string]*watchStat
	matcher FileMatcher
}

func NewTimerCheckFsNotifyWatcher(interval time.Duration, matcher FileMatcher) (FsNotifyWatcher, error) {
	w := &TimerCheckFsNotifyWatcher{
		mu:      sync.Mutex{},
		once:    sync.Once{},
		done:    make(chan struct{}),
		events:  make(chan fsnotify.Event),
		errors:  make(chan error),
		dirs:    make(map[string]*watchStat, defaultMapSize),
		files:   make(map[string]*watchStat, defaultMapSize),
		matcher: matcher,
	}

	go w.startTimerCheck(interval)

	return w, nil
}

func (tcw *TimerCheckFsNotifyWatcher) Events() <-chan fsnotify.Event {
	return tcw.events
}

func (tcw *TimerCheckFsNotifyWatcher) Errors() <-chan error {
	return tcw.errors
}

func (tcw *TimerCheckFsNotifyWatcher) AddWatch(name string, flags uint32) error {
	tcw.mu.Lock()
	defer tcw.mu.Unlock()

	var (
		s  *watchStat
		ok bool
	)

	if IsDir(name) {
		s, ok = tcw.dirs[name]
		if !ok {
			s = &watchStat{modTime: time.Now()}
			tcw.dirs[name] = s
		}
	} else {
		s, ok = tcw.files[name]
		if !ok {
			s = &watchStat{modTime: time.Now()}
			tcw.files[name] = s
		}
	}

	s.watchCreate = flags&FileCreateEvents > 0
	s.watchCreate = flags&FileRenameEvents > 0
	s.watchDelete = flags&FileDeleteEvents > 0
	s.watchWrite = flags&FileWriteEvents > 0

	return nil
}

func (tcw *TimerCheckFsNotifyWatcher) Remove(name string) error {
	tcw.mu.Lock()
	defer tcw.mu.Unlock()

	delete(tcw.files, name)
	delete(tcw.dirs, name)

	return nil
}

func (tcw *TimerCheckFsNotifyWatcher) Close() error {
	tcw.once.Do(func() {
		close(tcw.done)
	})

	return nil
}

func (tcw *TimerCheckFsNotifyWatcher) startTimerCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-tcw.done:
			return
		case <-ticker.C:
			tcw.checkFiles()
			tcw.checkDirs()
		}
	}
}

func (tcw *TimerCheckFsNotifyWatcher) checkFiles() {
	for f, t := range tcw.files {
		info, err := os.Stat(f)
		if err != nil {
			delete(tcw.files, f)

			if os.IsNotExist(err) {
				tcw.events <- fsnotify.Event{
					Name: f,
					Op:   fsnotify.Remove,
				}

				continue
			}

			tcw.errors <- err

			continue
		}

		if info == nil {
			continue
		}

		if t.watchWrite && info.ModTime().After(t.modTime) {
			t.modTime = info.ModTime()
			tcw.events <- fsnotify.Event{
				Name: f,
				Op:   fsnotify.Write,
			}
		}
	}
}

func (tcw *TimerCheckFsNotifyWatcher) checkDirs() {
	for dir, t := range tcw.dirs {
		f, err := os.Open(dir)
		if err != nil {
			tcw.handleDirError(dir, err)

			continue
		}

		fileInfos, err := f.Readdir(-1)
		_ = f.Close()

		if err != nil {
			tcw.handleDirError(dir, err)

			continue
		}

		for _, info := range fileInfos {
			filePath, isDirPath, pathErr := unlink(filepath.Join(dir, info.Name()), info)
			if pathErr != nil {
				logger.Debugf("read file error：%v", pathErr)

				continue
			}

			if isDirPath {
				tcw.addDir(filePath, t)

				continue
			}

			tcw.addFile(filePath, t, info.ModTime())
		}
	}
}

func (tcw *TimerCheckFsNotifyWatcher) handleDirError(dir string, err error) {
	delete(tcw.dirs, dir)

	if os.IsNotExist(err) {
		tcw.events <- fsnotify.Event{
			Name: dir,
			Op:   fsnotify.Remove,
		}

		return
	}

	tcw.errors <- err
}

func (tcw *TimerCheckFsNotifyWatcher) addDir(dir string, t *watchStat) {
	if _, ok := tcw.dirs[dir]; ok {
		return
	}

	tcw.dirs[dir] = &watchStat{
		watchCreate: t.watchCreate,
		watchRename: t.watchRename,
		watchDelete: t.watchDelete,
		watchWrite:  t.watchWrite,
		modTime:     time.Now(),
	}
}

func (tcw *TimerCheckFsNotifyWatcher) addFile(path string, t *watchStat, modTime time.Time) {
	if _, ok := tcw.files[path]; ok {
		return
	}

	tcw.files[path] = &watchStat{
		watchCreate: t.watchCreate,
		watchRename: t.watchRename,
		watchDelete: t.watchDelete,
		watchWrite:  t.watchWrite,
		modTime:     modTime,
	}

	tcw.events <- fsnotify.Event{
		Name: path,
		Op:   fsnotify.Create,
	}
}
