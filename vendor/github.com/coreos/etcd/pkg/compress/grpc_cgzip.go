// Copyright 2016 CoreOS, Inc.
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

package compress

import (
	"io"
	"io/ioutil"

	"github.com/youtube/vitess/go/cgzip"
)

// NewCGzipCompressor creates a Compressor based on github.com/youtube/vitess/go/cgzip.
func NewCGzipCompressor() *CGzipCompressor {
	return &CGzipCompressor{}
}

type CGzipCompressor struct{}

func (c *CGzipCompressor) Do(w io.Writer, p []byte) error {
	writer, err := cgzip.NewWriterLevelBuffer(w, cgzip.Z_BEST_SPEED, cgzip.DEFAULT_COMPRESSED_BUFFER_SIZE)
	if err != nil {
		return err
	}
	if _, err := writer.Write(p); err != nil {
		return err
	}
	return writer.Close()
}

func (c *CGzipCompressor) Type() string {
	return headers[CGzip]
}

// NewCGzipDecompressor creates a Decompressor based on github.com/youtube/vitess/go/cgzip.
func NewCGzipDecompressor() *CGzipDecompressor {
	return &CGzipDecompressor{}
}

type CGzipDecompressor struct{}

func (c *CGzipDecompressor) Do(r io.Reader) ([]byte, error) {
	readCloser, err := cgzip.NewReaderBuffer(r, cgzip.DEFAULT_COMPRESSED_BUFFER_SIZE)
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()
	return ioutil.ReadAll(readCloser)
}

func (c *CGzipDecompressor) Type() string {
	return headers[CGzip]
}

// NewCGzipLv2Compressor creates a Compressor based on github.com/youtube/vitess/go/cgzip.
func NewCGzipLv2Compressor() *CGzipLv2Compressor {
	return &CGzipLv2Compressor{}
}

type CGzipLv2Compressor struct{}

func (c *CGzipLv2Compressor) Do(w io.Writer, p []byte) error {
	writer, err := cgzip.NewWriterLevelBuffer(w, 2, cgzip.DEFAULT_COMPRESSED_BUFFER_SIZE)
	if err != nil {
		return err
	}
	if _, err := writer.Write(p); err != nil {
		return err
	}
	return writer.Close()
}

func (c *CGzipLv2Compressor) Type() string {
	return headers[CGzipLv2]
}

// NewCGzipLv2Decompressor creates a Decompressor based on github.com/youtube/vitess/go/cgzip.
func NewCGzipLv2Decompressor() *CGzipLv2Decompressor {
	return &CGzipLv2Decompressor{}
}

type CGzipLv2Decompressor struct{}

func (c *CGzipLv2Decompressor) Do(r io.Reader) ([]byte, error) {
	readCloser, err := cgzip.NewReaderBuffer(r, cgzip.DEFAULT_COMPRESSED_BUFFER_SIZE)
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()
	return ioutil.ReadAll(readCloser)
}

func (c *CGzipLv2Decompressor) Type() string {
	return headers[CGzipLv2]
}
