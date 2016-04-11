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

package control

import (
	"reflect"
	"testing"
)

func TestReadConfig(t *testing.T) {
	c, err := ReadConfig("test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Database != "etcdv3" {
		t.Fatalf("unexpected %s", c.Database)
	}
	if c.TestName != "bench-01-etcdv3" {
		t.Fatalf("unexpected %s", c.TestName)
	}
	if !reflect.DeepEqual(c.PeerIPs, []string{"10.240.0.13", "10.240.0.14", "10.240.0.15"}) {
		t.Fatalf("unexpected %s", c.PeerIPs)
	}
	if c.AgentPort != 3500 {
		t.Fatalf("unexpected %d", c.AgentPort)
	}
	if c.DatabasePort != 2379 {
		t.Fatalf("unexpected %d", c.DatabasePort)
	}
	if c.GoogleCloudProjectName != "etcd-development" {
		t.Fatalf("unexpected %s", c.GoogleCloudProjectName)
	}
	if c.GoogleCloudStorageKeyPath != "$HOME/gcloud-key.json" {
		t.Fatalf("unexpected %s", c.GoogleCloudStorageKeyPath)
	}
	if c.GoogleCloudStorageBucketName != "bench-20160411" {
		t.Fatalf("unexpected %s", c.GoogleCloudStorageBucketName)
	}
	if c.Step1.Skip {
		t.Fatalf("unexpected %v", c.Step1.Skip)
	}
	if c.Step1.DatabaseLogPath != "database.log" {
		t.Fatalf("unexpected %s", c.Step1.DatabaseLogPath)
	}
	if c.Step1.MonitorLogPath != "monitor.csv" {
		t.Fatalf("unexpected %s", c.Step1.MonitorLogPath)
	}
	if c.Step1.ZookeeperMaxClientCnxns != 5000 {
		t.Fatalf("unexpected %d", c.Step1.ZookeeperMaxClientCnxns)
	}
	if c.Step2.Skip {
		t.Fatalf("unexpected %v", c.Step2.Skip)
	}
	if c.Step2.BenchType != "write" {
		t.Fatalf("unexpected %s", c.Step2.BenchType)
	}
	if c.Step2.ResultPath != "bench-01-etcdv3-timeseries.csv" {
		t.Fatalf("unexpected %s", c.Step2.ResultPath)
	}
	if c.Step2.Connections != 100 {
		t.Fatalf("unexpected %d", c.Step2.Connections)
	}
	if !c.Step2.LocalRead {
		t.Fatalf("unexpected %v", c.Step2.LocalRead)
	}
	if c.Step2.TotalRequests != 3000000 {
		t.Fatalf("unexpected %d", c.Step2.TotalRequests)
	}
	if c.Step3.Skip {
		t.Fatalf("unexpected %v", c.Step3.Skip)
	}
}
