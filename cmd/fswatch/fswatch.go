package main

import (
	"flag"

	"github.com/vogo/fsnotify"
	"github.com/vogo/fwatch"
	"github.com/vogo/logger"
)

func main() {
	f := flag.String("f", "", "file to watch")

	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		_ = watcher.Close()
	}()

	if err = watcher.AddWatch(*f, fwatch.FileWriteRemoveEvents); err != nil {
		logger.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Warnf("failed to listen watch event")

				return
			}

			logger.Infof("event: %v", event)
		case err, ok := <-watcher.Errors:
			if !ok {
				logger.Warnf("failed to listen error event")

				return
			}

			logger.Errorf("watch error: %v", err)
		}
	}
}
