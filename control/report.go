// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package control

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/coreos/dbtester/remotestorage"
)

const (
	barChar = "∎"
)

type result struct {
	errStr   string
	duration time.Duration
	happened time.Time
}

type report struct {
	results   chan result
	sps       *secondPoints
	errorDist map[string]int
}

func printReport(results chan result, cfg Config) <-chan struct{} {
	return wrapReport(func() {
		r := &report{
			results:   results,
			errorDist: make(map[string]int),
			sps:       newSecondPoints(),
		}
		r.finalize()

		r.printSecondSample(cfg)

		if len(r.errorDist) > 0 {
			r.printErrors()
		}
	})
}

func wrapReport(f func()) <-chan struct{} {
	donec := make(chan struct{})
	go func() {
		defer close(donec)
		f()
	}()
	return donec
}

func (r *report) finalize() {
	log.Printf("finalize has started")
	st := time.Now()
	for res := range r.results {
		if res.errStr != "" {
			r.errorDist[res.errStr]++
		} else {
			r.sps.Add(res.happened, res.duration)
		}
	}
	log.Println("finalize took:", time.Since(st))
}

func (r *report) printSecondSample(cfg Config) {
	txt := r.sps.getTimeSeries().String()
	fmt.Println(txt)

	if err := toFile(txt, cfg.Step2.ResultPath); err != nil {
		log.Fatal(err)
	}

	log.Println("time series saved... Uploading to Google cloud storage...")
	u, err := remotestorage.NewGoogleCloudStorage([]byte(cfg.GoogleCloudStorageKey), cfg.GoogleCloudProjectName)
	if err != nil {
		log.Fatal(err)
	}

	srcCSVResultPath := cfg.Step2.ResultPath
	dstCSVResultPath := filepath.Base(cfg.Step2.ResultPath)
	log.Printf("Uploading %s to %s", srcCSVResultPath, dstCSVResultPath)

	var uerr error
	for k := 0; k < 5; k++ {
		if uerr = u.UploadFile(cfg.GoogleCloudStorageBucketName, srcCSVResultPath, dstCSVResultPath); uerr != nil {
			log.Println(uerr)
			continue
		} else {
			break
		}
	}
}

func (r *report) printErrors() {
	fmt.Printf("\nError distribution:\n")
	for err, num := range r.errorDist {
		fmt.Printf("  [%d]\t%s\n", num, err)
	}
}
