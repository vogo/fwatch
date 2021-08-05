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
	"os"
	"path/filepath"
)

func IsDir(name string) bool {
	stat, err := os.Stat(name)

	return err == nil && stat != nil && stat.IsDir()
}

func unlink(path string, info os.FileInfo) (unlinkPath string, dir bool, fileErr error) {
	if info.IsDir() {
		return path, true, nil
	}

	var err error

	for info.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)

		if err != nil {
			return "", false, err
		}

		info, err = os.Lstat(path)

		if err != nil {
			return "", false, err
		}
	}

	return path, info.IsDir(), nil
}
