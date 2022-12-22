// Copyright 2017 CoreOS, Inc.
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

package fileinspect

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func openToOverwrite(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func writeData(fpath string, data []byte) (n int, err error) {
	f, err := openToOverwrite(fpath)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return -1, err
		}
	}
	defer f.Close()
	n, err = f.Write(data)
	return
}

func createData() (dir string, n int64, err error) {
	dir, err = os.MkdirTemp(os.TempDir(), "fileinspect-write-test")
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		dirpath := filepath.Join(dir, fmt.Sprint(i))
		if err = os.MkdirAll(dirpath, 0777); err != nil {
			return
		}
		for j := 0; j < 10; j++ {
			fpath := filepath.Join(dirpath, fmt.Sprint(j))
			var written int
			if written, err = writeData(fpath, bytes.Repeat([]byte("a"), j)); err != nil {
				return
			}
			n += int64(written)
		}
	}
	return
}

func TestWalkSize(t *testing.T) {
	dir, n, err := createData()
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("walking %q (written %d bytes)\n", dir, n)
	for k, v := range fm {
		fmt.Printf("%q : %+v\n", k, v)
	}

	size, err := Size(dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != size {
		t.Fatalf("size expected %d, got %d", n, size)
	}
}
