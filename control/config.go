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
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Database                       string `yaml:"database"`
	TestName                       string `yaml:"test_name"`
	GoogleCloudProjectName         string `yaml:"google_cloud_project_name"`
	GoogleCloudStorageKey          string
	GoogleCloudStorageKeyPath      string `yaml:"google_cloud_storage_key_path"`
	GoogleCloudStorageBucketName   string `yaml:"google_cloud_storage_bucket_name"`
	GoogleCloudStorageSubDirectory string `yaml:"google_cloud_storage_sub_directory"`

	PeerIPs      []string `yaml:"peer_ips"`
	PeerIPString string
	AgentPort    int `yaml:"agent_port"`
	DatabasePort int `yaml:"database_port"`

	AgentEndpoints    []string
	DatabaseEndpoints []string

	// cgzip, cgzip-lv2, gzip, snappy, snappy-cpp
	EtcdCompression string `yaml:"etcd_compression"`

	Step1 struct {
		Skip bool `yaml:"skip"`

		ZookeeperMaxClientCnxns int64 `yaml:"zookeeper_max_client_connections"`
		ZookeeperSnapCount      int64 `yaml:"zookeeper_snap_count"`
	} `yaml:"step1"`

	Step2 struct {
		Skip                  bool   `yaml:"skip"`
		BenchType             string `yaml:"bench_type"`
		LocalRead             bool   `yaml:"local_read"`
		ResultPath            string `yaml:"result_path"`
		Connections           int    `yaml:"connections"`
		Clients               int    `yaml:"clients"`
		KeySize               int    `yaml:"key_size"`
		ValueSize             int    `yaml:"value_size"`
		TotalRequests         int    `yaml:"total_requests"`
		Etcdv3CompactionCycle int    `yaml:"etcdv3_compaction_cycle"`
	} `yaml:"step2"`

	Step3 struct {
		Skip       bool   `yaml:"skip"`
		ResultPath string `yaml:"result_path"`
	}
}

func ReadConfig(fpath string) (Config, error) {
	bts, err := ioutil.ReadFile(fpath)
	if err != nil {
		return Config{}, err
	}
	rs := Config{}
	if err := yaml.Unmarshal(bts, &rs); err != nil {
		return Config{}, err
	}
	return rs, nil
}
