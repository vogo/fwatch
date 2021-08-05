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

package fwatch_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkFile(t *testing.T) {
	t.Parallel()

	linkDir := filepath.Join(os.TempDir(), "fwatch-util-link")
	tempDir := filepath.Join(os.TempDir(), "fwatch-util")
	_ = os.Mkdir(linkDir, os.ModePerm)
	_ = os.Mkdir(tempDir, os.ModePerm)

	defer func() {
		_ = os.RemoveAll(tempDir)
		_ = os.RemoveAll(linkDir)
	}()

	linkDirInTemp := filepath.Join(tempDir, "link-dir")
	_ = os.Symlink(linkDir, linkDirInTemp)

	info, err := os.Lstat(linkDirInTemp)
	if err != nil {
		t.Error(err)
	}

	t.Log("linkDir name:", info.Name())
	t.Log("linkDir is dir:", info.IsDir())
	t.Log("linkDir file mode:", info.Mode())
	t.Log("linkDir is link:", info.Mode()&os.ModeSymlink != 0)
	t.Log(filepath.EvalSymlinks(linkDirInTemp))
}
