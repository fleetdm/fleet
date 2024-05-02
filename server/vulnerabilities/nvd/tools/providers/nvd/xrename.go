// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nvd

import (
	"fmt"
	"io"
	"os"
)

// xRename tries to rename oldpath to newpath, if it gets LinkError (most often
// because of the files located on a different device) it copies and removes
// it instead
func xRename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if _, ok := err.(*os.LinkError); ok {
		var oldfile, newfile *os.File
		if oldfile, err = os.Open(oldpath); err != nil {
			return err
		}
		defer oldfile.Close()
		var finfo os.FileInfo
		if finfo, err = oldfile.Stat(); err != nil {
			return err
		}
		if !finfo.Mode().IsRegular() {
			return fmt.Errorf("failed to rename %q to %q: source file is not a regular file", oldpath, newpath)
		}
		if newfile, err = os.OpenFile(newpath, os.O_WRONLY|os.O_CREATE, finfo.Mode().Perm()); err != nil {
			return err
		}
		defer newfile.Close()
		if _, err = io.Copy(newfile, oldfile); err != nil {
			return err
		}
		err = os.Remove(oldpath)
	}
	return err
}
