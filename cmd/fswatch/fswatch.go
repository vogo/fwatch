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

	"github.com/vogo/fsnotify"
	"github.com/vogo/fwatch"
	"github.com/vogo/logger"
)

func main() {
	filePath := flag.String("f", "", "file to watch")

	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		_ = watcher.Close()
	}()

	if err = watcher.AddWatch(*filePath, fwatch.FileWriteRemoveEvents); err != nil {
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
