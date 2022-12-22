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

package dbtester

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/etcd-io/dbtester/dbtesterpb"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// MakeTag converts database description to database tag.
func MakeTag(desc string) string {
	s := strings.ToLower(desc)
	s = strings.Replace(s, "go ", "go", -1)
	s = strings.Replace(s, "java ", "java", -1)
	s = strings.Replace(s, "(", "", -1)
	s = strings.Replace(s, ")", "", -1)
	return strings.Replace(s, " ", "-", -1)
}

// Config configures dbtester control clients.
type Config struct {
	lg *zap.Logger

	TestTitle       string `yaml:"test_title"`
	TestDescription string `yaml:"test_description"`

	dbtesterpb.ConfigClientMachineInitial `yaml:"config_client_machine_initial"`

	AllDatabaseIDList                           []string                                              `yaml:"all_database_id_list"`
	DatabaseIDToConfigClientMachineAgentControl map[string]dbtesterpb.ConfigClientMachineAgentControl `yaml:"datatbase_id_to_config_client_machine_agent_control"`
	DatabaseIDToConfigAnalyzeMachineInitial     map[string]dbtesterpb.ConfigAnalyzeMachineInitial     `yaml:"datatbase_id_to_config_analyze_machine_initial"`

	dbtesterpb.ConfigAnalyzeMachineAllAggregatedOutput `yaml:"analyze_all_aggregated_output"`
	AnalyzePlotPathPrefix                              string                                `yaml:"analyze_plot_path_prefix"`
	AnalyzePlotList                                    []dbtesterpb.ConfigAnalyzeMachinePlot `yaml:"analyze_plot_list"`
	dbtesterpb.ConfigAnalyzeMachineREADME              `yaml:"analyze_readme"`
}

// ReadConfig reads control configuration file.
func ReadConfig(fpath string, analyze bool) (*Config, error) {
	bts, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	if err = yaml.Unmarshal(bts, &cfg); err != nil {
		return nil, err
	}

	lg, lerr := zap.NewProduction()
	if lerr != nil {
		return nil, lerr
	}
	cfg.lg = lg

	for _, id := range cfg.AllDatabaseIDList {
		if !dbtesterpb.IsValidDatabaseID(id) {
			return nil, fmt.Errorf("databaseID %q is unknown", id)
		}
	}

	if cfg.ConfigClientMachineInitial.PathPrefix != "" {
		cfg.ConfigClientMachineInitial.LogPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.LogPath)
		cfg.ConfigClientMachineInitial.ClientSystemMetricsPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientSystemMetricsPath)
		cfg.ConfigClientMachineInitial.ClientSystemMetricsInterpolatedPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientSystemMetricsInterpolatedPath)
		cfg.ConfigClientMachineInitial.ClientLatencyThroughputTimeseriesPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientLatencyThroughputTimeseriesPath)
		cfg.ConfigClientMachineInitial.ClientLatencyDistributionAllPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientLatencyDistributionAllPath)
		cfg.ConfigClientMachineInitial.ClientLatencyDistributionPercentilePath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientLatencyDistributionPercentilePath)
		cfg.ConfigClientMachineInitial.ClientLatencyDistributionSummaryPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientLatencyDistributionSummaryPath)
		cfg.ConfigClientMachineInitial.ClientLatencyByKeyNumberPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ClientLatencyByKeyNumberPath)
		cfg.ConfigClientMachineInitial.ServerDiskSpaceUsageSummaryPath = filepath.Join(cfg.ConfigClientMachineInitial.PathPrefix, cfg.ConfigClientMachineInitial.ServerDiskSpaceUsageSummaryPath)
	}

	for databaseID, group := range cfg.DatabaseIDToConfigClientMachineAgentControl {
		if !dbtesterpb.IsValidDatabaseID(databaseID) {
			return nil, fmt.Errorf("databaseID %q is unknown", databaseID)
		}

		group.DatabaseID = databaseID
		group.DatabaseTag = MakeTag(group.DatabaseDescription)
		group.PeerIPsString = strings.Join(group.PeerIPs, "___")
		group.DatabaseEndpoints = make([]string, len(group.PeerIPs))
		group.AgentEndpoints = make([]string, len(group.PeerIPs))
		for j := range group.PeerIPs {
			group.DatabaseEndpoints[j] = fmt.Sprintf("%s:%d", group.PeerIPs[j], group.DatabasePortToConnect)
			group.AgentEndpoints[j] = fmt.Sprintf("%s:%d", group.PeerIPs[j], group.AgentPortToConnect)
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[databaseID] = group
	}

	for databaseID, amc := range cfg.DatabaseIDToConfigAnalyzeMachineInitial {
		amc.PathPrefix = strings.TrimSpace(amc.PathPrefix)
		amc.DatabaseID = databaseID
		amc.DatabaseTag = cfg.DatabaseIDToConfigClientMachineAgentControl[databaseID].DatabaseTag
		amc.DatabaseDescription = cfg.DatabaseIDToConfigClientMachineAgentControl[databaseID].DatabaseDescription

		if amc.PathPrefix != "" {
			amc.ClientSystemMetricsInterpolatedPath = amc.PathPrefix + "-" + amc.ClientSystemMetricsInterpolatedPath
			amc.ClientLatencyThroughputTimeseriesPath = amc.PathPrefix + "-" + amc.ClientLatencyThroughputTimeseriesPath
			amc.ClientLatencyDistributionAllPath = amc.PathPrefix + "-" + amc.ClientLatencyDistributionAllPath
			amc.ClientLatencyDistributionPercentilePath = amc.PathPrefix + "-" + amc.ClientLatencyDistributionPercentilePath
			amc.ClientLatencyDistributionSummaryPath = amc.PathPrefix + "-" + amc.ClientLatencyDistributionSummaryPath
			amc.ClientLatencyByKeyNumberPath = amc.PathPrefix + "-" + amc.ClientLatencyByKeyNumberPath
			amc.ServerDiskSpaceUsageSummaryPath = amc.PathPrefix + "-" + amc.ServerDiskSpaceUsageSummaryPath
			amc.ServerMemoryByKeyNumberPath = amc.PathPrefix + "-" + amc.ServerMemoryByKeyNumberPath
			amc.ServerReadBytesDeltaByKeyNumberPath = amc.PathPrefix + "-" + amc.ServerReadBytesDeltaByKeyNumberPath
			amc.ServerWriteBytesDeltaByKeyNumberPath = amc.PathPrefix + "-" + amc.ServerWriteBytesDeltaByKeyNumberPath
			for i := range amc.ServerSystemMetricsInterpolatedPathList {
				amc.ServerSystemMetricsInterpolatedPathList[i] = amc.PathPrefix + "-" + amc.ServerSystemMetricsInterpolatedPathList[i]
			}
			amc.AllAggregatedOutputPath = amc.PathPrefix + "-" + amc.AllAggregatedOutputPath
		}

		cfg.DatabaseIDToConfigAnalyzeMachineInitial[databaseID] = amc
	}

	for databaseID, ctrl := range cfg.DatabaseIDToConfigClientMachineAgentControl {
		if databaseID != dbtesterpb.DatabaseID_etcd__other.String() &&
			databaseID != dbtesterpb.DatabaseID_etcd__tip.String() &&
			databaseID != dbtesterpb.DatabaseID_etcd__v3_2.String() &&
			databaseID != dbtesterpb.DatabaseID_etcd__v3_3.String() &&
			ctrl.ConfigClientMachineBenchmarkOptions.ConnectionNumber != ctrl.ConfigClientMachineBenchmarkOptions.ClientNumber {
			return nil, fmt.Errorf("%q got connected %d != clients %d", databaseID, ctrl.ConfigClientMachineBenchmarkOptions.ConnectionNumber, ctrl.ConfigClientMachineBenchmarkOptions.ClientNumber)
		}
	}

	const (
		defaultAgentPort           int64 = 3500
		defaultEtcdClientPort      int64 = 2379
		defaultZookeeperClientPort int64 = 2181
		defaultConsulClientPort    int64 = 8500

		defaultEtcdSnapshotCount             int64 = 100000
		defaultEtcdQuotaSizeBytes            int64 = 8000000000
		defaultZookeeperSnapCount            int64 = 100000
		defaultZookeeperTickTime             int64 = 2000
		defaultZookeeperInitLimit            int64 = 5
		defaultZookeeperSyncLimit            int64 = 5
		defaultZookeeperMaxClientConnections int64 = 5000
	)

	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__other.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultEtcdClientPort
		}
		if v.Flag_Etcd_Other.SnapshotCount == 0 {
			v.Flag_Etcd_Other.SnapshotCount = defaultEtcdSnapshotCount
		}
		if v.Flag_Etcd_Other.QuotaSizeBytes == 0 {
			v.Flag_Etcd_Other.QuotaSizeBytes = defaultEtcdQuotaSizeBytes
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__other.String()] = v
	}
	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__tip.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultEtcdClientPort
		}
		if v.Flag_Etcd_Tip.SnapshotCount == 0 {
			v.Flag_Etcd_Tip.SnapshotCount = defaultEtcdSnapshotCount
		}
		if v.Flag_Etcd_Tip.QuotaSizeBytes == 0 {
			v.Flag_Etcd_Tip.QuotaSizeBytes = defaultEtcdQuotaSizeBytes
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__tip.String()] = v
	}
	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_2.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultEtcdClientPort
		}
		if v.Flag_Etcd_V3_2.SnapshotCount == 0 {
			v.Flag_Etcd_V3_2.SnapshotCount = defaultEtcdSnapshotCount
		}
		if v.Flag_Etcd_V3_2.QuotaSizeBytes == 0 {
			v.Flag_Etcd_V3_2.QuotaSizeBytes = defaultEtcdQuotaSizeBytes
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_2.String()] = v
	}
	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_3.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultEtcdClientPort
		}
		if v.Flag_Etcd_V3_3.SnapshotCount == 0 {
			v.Flag_Etcd_V3_3.SnapshotCount = defaultEtcdSnapshotCount
		}
		if v.Flag_Etcd_V3_3.QuotaSizeBytes == 0 {
			v.Flag_Etcd_V3_3.QuotaSizeBytes = defaultEtcdQuotaSizeBytes
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_3.String()] = v
	}

	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_zookeeper__r3_5_3_beta.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultZookeeperClientPort
		}
		v.Flag_Zookeeper_R3_5_3Beta.ClientPort = v.DatabasePortToConnect
		if v.Flag_Zookeeper_R3_5_3Beta.TickTime == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.TickTime = defaultZookeeperTickTime
		}
		if v.Flag_Zookeeper_R3_5_3Beta.TickTime == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.TickTime = defaultZookeeperTickTime
		}
		if v.Flag_Zookeeper_R3_5_3Beta.InitLimit == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.InitLimit = defaultZookeeperInitLimit
		}
		if v.Flag_Zookeeper_R3_5_3Beta.SyncLimit == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.SyncLimit = defaultZookeeperSyncLimit
		}
		if v.Flag_Zookeeper_R3_5_3Beta.SnapCount == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.SnapCount = defaultZookeeperSnapCount
		}
		if v.Flag_Zookeeper_R3_5_3Beta.MaxClientConnections == 0 {
			v.Flag_Zookeeper_R3_5_3Beta.MaxClientConnections = defaultZookeeperMaxClientConnections
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_zookeeper__r3_5_3_beta.String()] = v
	}

	if v, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_consul__v1_0_2.String()]; ok {
		if v.AgentPortToConnect == 0 {
			v.AgentPortToConnect = defaultAgentPort
		}
		if v.DatabasePortToConnect == 0 {
			v.DatabasePortToConnect = defaultConsulClientPort
		}
		cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_consul__v1_0_2.String()] = v
	}

	// need etcd configs since it's backed by etcd
	if _, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_zetcd__beta.String()]; ok {
		_, okOther := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__other.String()]
		_, okTip := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__tip.String()]
		_, ok32 := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_2.String()]
		_, ok33 := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_3.String()]
		if !okOther && !okTip && !ok32 && !ok33 {
			return nil, fmt.Errorf("got %q config, but no etcd config is given", dbtesterpb.DatabaseID_zetcd__beta.String())
		}
	}
	if _, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_cetcd__beta.String()]; ok {
		_, okOther := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__other.String()]
		_, okTip := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__tip.String()]
		_, ok32 := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_2.String()]
		_, ok33 := cfg.DatabaseIDToConfigClientMachineAgentControl[dbtesterpb.DatabaseID_etcd__v3_3.String()]
		if !okOther && !okTip && !ok32 && !ok33 {
			return nil, fmt.Errorf("got %q config, but no etcd config is given", dbtesterpb.DatabaseID_cetcd__beta.String())
		}
	}

	if cfg.ConfigClientMachineInitial.GoogleCloudStorageKeyPath != "" && !analyze {
		bts, err = os.ReadFile(cfg.ConfigClientMachineInitial.GoogleCloudStorageKeyPath)
		if err != nil {
			return nil, err
		}
		cfg.ConfigClientMachineInitial.GoogleCloudStorageKey = string(bts)
	}

	for i := range cfg.AnalyzePlotList {
		cfg.AnalyzePlotList[i].OutputPathCSV = filepath.Join(cfg.AnalyzePlotPathPrefix, cfg.AnalyzePlotList[i].Column+".csv")
		cfg.AnalyzePlotList[i].OutputPathList = make([]string, 2)
		cfg.AnalyzePlotList[i].OutputPathList[0] = filepath.Join(cfg.AnalyzePlotPathPrefix, cfg.AnalyzePlotList[i].Column+".svg")
		cfg.AnalyzePlotList[i].OutputPathList[1] = filepath.Join(cfg.AnalyzePlotPathPrefix, cfg.AnalyzePlotList[i].Column+".png")
	}

	return &cfg, nil
}

const maxEtcdQuotaSize = 8000000000

// ToRequest converts configuration to 'dbtesterpb.Request'.
func (cfg *Config) ToRequest(databaseID string, op dbtesterpb.Operation, idx int) (req *dbtesterpb.Request, err error) {
	gcfg, ok := cfg.DatabaseIDToConfigClientMachineAgentControl[databaseID]
	if !ok {
		err = fmt.Errorf("database ID %q is not defined", databaseID)
		return
	}
	did := dbtesterpb.DatabaseID(dbtesterpb.DatabaseID_value[databaseID])

	req = &dbtesterpb.Request{
		Operation:           op,
		TriggerLogUpload:    gcfg.ConfigClientMachineBenchmarkSteps.Step4UploadLogs,
		DatabaseID:          did,
		DatabaseTag:         gcfg.DatabaseTag,
		PeerIPsString:       gcfg.PeerIPsString,
		IPIndex:             uint32(idx),
		CurrentClientNumber: gcfg.ConfigClientMachineBenchmarkOptions.ClientNumber,
		ConfigClientMachineInitial: &dbtesterpb.ConfigClientMachineInitial{
			GoogleCloudProjectName:         cfg.ConfigClientMachineInitial.GoogleCloudProjectName,
			GoogleCloudStorageKey:          cfg.ConfigClientMachineInitial.GoogleCloudStorageKey,
			GoogleCloudStorageBucketName:   cfg.ConfigClientMachineInitial.GoogleCloudStorageBucketName,
			GoogleCloudStorageSubDirectory: cfg.ConfigClientMachineInitial.GoogleCloudStorageSubDirectory,
		},
	}

	switch req.DatabaseID {
	case dbtesterpb.DatabaseID_etcd__other:
		if gcfg.Flag_Etcd_Other.QuotaSizeBytes > maxEtcdQuotaSize {
			err = fmt.Errorf("maximum etcd quota is 8 GB (%d), got %d", maxEtcdQuotaSize, gcfg.Flag_Etcd_Other.QuotaSizeBytes)
			return
		}
		req.Flag_Etcd_Other = &dbtesterpb.Flag_Etcd_Other{
			SnapshotCount:  gcfg.Flag_Etcd_Other.SnapshotCount,
			QuotaSizeBytes: gcfg.Flag_Etcd_Other.QuotaSizeBytes,
		}
	case dbtesterpb.DatabaseID_etcd__tip:
		if gcfg.Flag_Etcd_Tip.QuotaSizeBytes > maxEtcdQuotaSize {
			err = fmt.Errorf("maximum etcd quota is 8 GB (%d), got %d", maxEtcdQuotaSize, gcfg.Flag_Etcd_Tip.QuotaSizeBytes)
			return
		}
		req.Flag_Etcd_Tip = &dbtesterpb.Flag_Etcd_Tip{
			SnapshotCount:  gcfg.Flag_Etcd_Tip.SnapshotCount,
			QuotaSizeBytes: gcfg.Flag_Etcd_Tip.QuotaSizeBytes,
		}
	case dbtesterpb.DatabaseID_etcd__v3_2:
		if gcfg.Flag_Etcd_V3_2.QuotaSizeBytes > maxEtcdQuotaSize {
			err = fmt.Errorf("maximum etcd quota is 8 GB (%d), got %d", maxEtcdQuotaSize, gcfg.Flag_Etcd_V3_2.QuotaSizeBytes)
			return
		}
		req.Flag_Etcd_V3_2 = &dbtesterpb.Flag_Etcd_V3_2{
			SnapshotCount:  gcfg.Flag_Etcd_V3_2.SnapshotCount,
			QuotaSizeBytes: gcfg.Flag_Etcd_V3_2.QuotaSizeBytes,
		}
	case dbtesterpb.DatabaseID_etcd__v3_3:
		if gcfg.Flag_Etcd_V3_3.QuotaSizeBytes > maxEtcdQuotaSize {
			err = fmt.Errorf("maximum etcd quota is 8 GB (%d), got %d", maxEtcdQuotaSize, gcfg.Flag_Etcd_V3_3.QuotaSizeBytes)
			return
		}
		req.Flag_Etcd_V3_3 = &dbtesterpb.Flag_Etcd_V3_3{
			SnapshotCount:  gcfg.Flag_Etcd_V3_3.SnapshotCount,
			QuotaSizeBytes: gcfg.Flag_Etcd_V3_3.QuotaSizeBytes,
		}

	case dbtesterpb.DatabaseID_zookeeper__r3_5_3_beta:
		req.Flag_Zookeeper_R3_5_3Beta = &dbtesterpb.Flag_Zookeeper_R3_5_3Beta{
			JavaDJuteMaxBuffer:   gcfg.Flag_Zookeeper_R3_5_3Beta.JavaDJuteMaxBuffer,
			JavaXms:              gcfg.Flag_Zookeeper_R3_5_3Beta.JavaXms,
			JavaXmx:              gcfg.Flag_Zookeeper_R3_5_3Beta.JavaXmx,
			MyID:                 uint32(idx + 1),
			ClientPort:           gcfg.Flag_Zookeeper_R3_5_3Beta.ClientPort,
			TickTime:             gcfg.Flag_Zookeeper_R3_5_3Beta.TickTime,
			InitLimit:            gcfg.Flag_Zookeeper_R3_5_3Beta.InitLimit,
			SyncLimit:            gcfg.Flag_Zookeeper_R3_5_3Beta.SyncLimit,
			SnapCount:            gcfg.Flag_Zookeeper_R3_5_3Beta.SnapCount,
			MaxClientConnections: gcfg.Flag_Zookeeper_R3_5_3Beta.MaxClientConnections,
		}

	case dbtesterpb.DatabaseID_consul__v1_0_2:

	case dbtesterpb.DatabaseID_zetcd__beta:
	case dbtesterpb.DatabaseID_cetcd__beta:

	default:
		err = fmt.Errorf("unknown %v", req.DatabaseID)
	}

	return
}
