package main

import (
	"flag"
	"strings"
	"time"

	"github.com/vogo/fwatch"
	"github.com/vogo/logger"
)

const (
	defaultInactiveSeconds = 60
	defaultSilenceSeconds  = 60 * 8
)

func main() {
	var (
		dir             = flag.String("dir", "", "directory to watch")
		method          = flag.String("method", "", "watch method, os/timer")
		logLevel        = flag.String("log_level", "", "log level(debug/info)")
		includeSub      = flag.Bool("include_sub", false, "whether include sub-directories")
		fileSuffix      = flag.String("suffix", "", "file suffix to watch")
		inactiveSeconds = flag.Int64("inactive_seconds", defaultInactiveSeconds, "after seconds files is inactive")
		silenceSeconds  = flag.Int64("silence_seconds", defaultSilenceSeconds, "after seconds files is silence")
	)

	flag.Parse()

	if *dir == "" {
		logger.Fatal("required parameter -dir")
	}

	if strings.EqualFold(*logLevel, "DEBUG") {
		logger.SetLevel(logger.LevelDebug)
	}

	var watchMethod interface{} = *method

	inactiveDuration := time.Duration(*inactiveSeconds) * time.Second
	silenceDuration := time.Duration(*silenceSeconds) * time.Second

	watcher, err := fwatch.NewFileWatcher(watchMethod.(fwatch.WatchMethod), inactiveDuration, silenceDuration)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		_ = watcher.Stop()
	}()

	go func() {
		for {
			select {
			case <-watcher.Stopper.C:
				return
			case watchErr := <-watcher.Errors:
				logger.Infof("--> error: %v", watchErr)
			case f := <-watcher.Events:
				logger.Infof("--> event: %v", f)
			}
		}
	}()

	if dirErr := watcher.WatchDir(*dir, *includeSub, func(s string) bool {
		return *fileSuffix == "" || strings.HasSuffix(s, *fileSuffix)
	}); dirErr != nil {
		logger.Fatal(dirErr)
	}

	select {}
}
