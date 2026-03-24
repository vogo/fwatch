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

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vogo/fwatch"
	"github.com/vogo/vogo/vlog"
)

const (
	defaultInactiveSeconds = 60
	defaultSilenceSeconds  = 60 * 8
)

func main() {
	var (
		file            = flag.String("file", "", "watch a single file for changes")
		dir             = flag.String("dir", "", "watch a directory for file changes")
		method          = flag.String("method", "timer", "watch method: fs (OS-level fsnotify) or timer (polling)")
		logLevel        = flag.String("log_level", "", "log level: debug or info")
		includeSub      = flag.Bool("include_sub", false, "include sub-directories when watching a directory")
		fileSuffix      = flag.String("suffix", "", "only watch files with this suffix (e.g. .log)")
		inactiveSeconds = flag.Int64("inactive_seconds", defaultInactiveSeconds, "seconds before a file is considered inactive")
		silenceSeconds  = flag.Int64("silence_seconds", defaultSilenceSeconds, "seconds before a file is removed from watch")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `fwatch - a command line file and directory watcher to show the functionality of fwatch library

Usage:
  fwatch -file <path>                     Watch a single file
  fwatch -dir <path> [options]            Watch a directory

Examples:
  fwatch -file /var/log/app.log
  fwatch -dir /var/log -method fs -include_sub -suffix .log
  fwatch -dir /tmp -inactive_seconds 30 -silence_seconds 120

Options:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *file == "" && *dir == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *file != "" && *dir != "" {
		fmt.Fprintln(os.Stderr, "error: -file and -dir cannot be used together")
		os.Exit(1)
	}

	if strings.EqualFold(*logLevel, "DEBUG") {
		vlog.SetLevel(vlog.LevelDebug)
	}

	if *file != "" {
		watchFile(*file)
	} else {
		watchDir(*dir, *method, *includeSub, *fileSuffix, *inactiveSeconds, *silenceSeconds)
	}
}

func watchFile(filePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		vlog.Fatal(err)
	}

	defer func() {
		_ = watcher.Close()
	}()

	if err = watcher.Add(filePath); err != nil {
		vlog.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				vlog.Warnf("failed to listen watch event")

				return
			}

			vlog.Infof("event: %v", event)
		case err, ok := <-watcher.Errors:
			if !ok {
				vlog.Warnf("failed to listen error event")

				return
			}

			vlog.Errorf("watch error: %v", err)
		}
	}
}

func watchDir(dir, method string, includeSub bool, fileSuffix string, inactiveSeconds, silenceSeconds int64) {
	inactiveDuration := time.Duration(inactiveSeconds) * time.Second
	silenceDuration := time.Duration(silenceSeconds) * time.Second

	watcher, err := fwatch.New(
		fwatch.WithMethod(fwatch.WatchMethod(method)),
		fwatch.WithInactiveDuration(inactiveDuration),
		fwatch.WithSilenceDuration(silenceDuration),
	)
	if err != nil {
		vlog.Fatal(err)
	}

	defer func() {
		_ = watcher.Stop()
	}()

	go func() {
		for {
			select {
			case <-watcher.Done():
				return
			case watchErr := <-watcher.Errors:
				vlog.Infof("--> error: %v", watchErr)
			case f := <-watcher.Events:
				vlog.Infof("--> event: %v", f)
			}
		}
	}()

	if dirErr := watcher.WatchDir(dir, includeSub, func(s string) bool {
		return fileSuffix == "" || strings.HasSuffix(s, fileSuffix)
	}); dirErr != nil {
		vlog.Fatal(dirErr)
	}

	select {}
}
