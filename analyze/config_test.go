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

package analyze

import "testing"

func TestReadConfig(t *testing.T) {
	c, err := ReadConfig("test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Step1[0].DataPathList[0] != "bench-20160330/bench-01-etcd-1-monitor.csv" {
		t.Fatalf("unexpected %s", c.Step1[0].DataPathList[0])
	}
	if c.Step2[0].DataList[0].Path != "bench-20160330/bench-01-etcd-aggregated.csv" {
		t.Fatalf("unexpected %s", c.Step2[0].DataList[0].Path)
	}
	if c.Step3[0].PlotList[0].Lines[0].Column != "avg_latency_ms_etcd_v3" {
		t.Fatalf("unexpected %s", c.Step3[0].PlotList[0].Lines[0].Column)
	}
	if c.Step3[0].PlotList[0].YAxis != "Latency(millisecond)" {
		t.Fatalf("unexpected %s", c.Step3[0].PlotList[0].YAxis)
	}
	if c.Step3[0].PlotList[0].OutputPathList[1] != "bench-20160330/bench-01-avg-latency-ms.png" {
		t.Fatalf("unexpected %s", c.Step3[0].PlotList[0].OutputPathList[1])
	}
	if c.Step4.OutputPath != "bench-20160330/README.md" {
		t.Fatalf("unexpected %s", c.Step4.OutputPath)
	}
	if c.Step4.Results[0].Images[0].ImageTitle != "bench-01-avg-latency-ms" {
		t.Fatalf("unexpected %s", c.Step4.Results[0].Images[0].ImageTitle)
	}
	if c.Step4.Results[0].Images[0].ImagePath != "bench-20160330/bench-01-avg-latency-ms.png" {
		t.Fatalf("unexpected %s", c.Step4.Results[0].Images[0].ImagePath)
	}
	if c.Step4.Results[0].Images[0].ImageType != "local" {
		t.Fatalf("unexpected %s", c.Step4.Results[0].Images[0].ImageType)
	}
}
