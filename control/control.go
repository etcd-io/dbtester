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
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/coreos/dbtester/agent"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type (
	Flags struct {
		Database                string
		AgentEndpoints          []string
		ZookeeperPreAllocSize   int64
		ZookeeperMaxClientCnxns int64

		LogPrefix              string
		DatabaseLogPath        string
		MonitorResultPath      string
		GoogleCloudProjectName string
		KeyPath                string
		Bucket                 string
	}
)

var (
	StartCommand = &cobra.Command{
		Use:   "start",
		Short: "Starts database through RPC calls.",
		RunE:  CommandFunc,
	}
	StopCommand = &cobra.Command{
		Use:   "stop",
		Short: "Stops database through RPC calls.",
		RunE:  CommandFunc,
	}
	RestartCommand = &cobra.Command{
		Use:   "restart",
		Short: "Restarts database through RPC calls.",
		RunE:  CommandFunc,
	}
	globalFlags = Flags{}
)

func init() {
	StartCommand.PersistentFlags().StringVar(&globalFlags.Database, "database", "", "etcdv2, etcdv3, zookeeper, zk, consul.")
	StartCommand.PersistentFlags().StringSliceVar(&globalFlags.AgentEndpoints, "agent-endpoints", []string{""}, "Endpoints to send client requests to, then it automatically configures.")
	StartCommand.PersistentFlags().Int64Var(&globalFlags.ZookeeperPreAllocSize, "zk-pre-alloc-size", 65536*1024, "Disk pre-allocation size in bytes.")
	StartCommand.PersistentFlags().Int64Var(&globalFlags.ZookeeperMaxClientCnxns, "zk-max-client-conns", 5000, "Maximum number of concurrent Zookeeper connection.")

	StartCommand.PersistentFlags().StringVar(&globalFlags.LogPrefix, "log-prefix", "", "Prefix to all logs to be generated in agents.")
	StartCommand.PersistentFlags().StringVar(&globalFlags.DatabaseLogPath, "database-log-path", "database.log", "Path of database log.")
	StartCommand.PersistentFlags().StringVar(&globalFlags.MonitorResultPath, "monitor-result-path", "monitor.csv", "CSV file path of monitoring results.")
	StartCommand.PersistentFlags().StringVar(&globalFlags.GoogleCloudProjectName, "google-cloud-project-name", "", "Google cloud project name.")
	StartCommand.PersistentFlags().StringVar(&globalFlags.KeyPath, "key-path", "", "Path of key file.")
	StartCommand.PersistentFlags().StringVar(&globalFlags.Bucket, "bucket", "", "Bucket name in cloud storage.")

	StopCommand.PersistentFlags().StringSliceVar(&globalFlags.AgentEndpoints, "agent-endpoints", []string{""}, "Endpoints to send client requests to, then it automatically configures.")

	RestartCommand.PersistentFlags().StringSliceVar(&globalFlags.AgentEndpoints, "agent-endpoints", []string{""}, "Endpoints to send client requests to, then it automatically configures.")
}

func CommandFunc(cmd *cobra.Command, args []string) error {
	if globalFlags.Database == "zk" {
		globalFlags.Database = "zookeeper"
	}
	req := agent.Request{}

	switch cmd.Use {
	case "start":
		req.Operation = agent.Request_Start
	case "stop":
		req.Operation = agent.Request_Stop
	case "restart":
		req.Operation = agent.Request_Restart
	default:
		return fmt.Errorf("Operation '%s' is not supported!\n", cmd.Use)
	}

	switch globalFlags.Database {
	case "etcdv2":
		req.Database = agent.Request_etcdv2
	case "etcdv3":
		req.Database = agent.Request_etcdv3
	case "zookeeper":
		req.Database = agent.Request_ZooKeeper
	case "consul":
		req.Database = agent.Request_Consul
	default:
		if req.Operation != agent.Request_Stop {
			return fmt.Errorf("'%s' is not supported!\n", globalFlags.Database)
		}
	}
	peerIPs := extractIPs(globalFlags.AgentEndpoints)
	req.PeerIPs = strings.Join(peerIPs, "___") // because protoc mixes the order of 'repeated' type data

	if cmd.Use == "start" {
		req.ZookeeperPreAllocSize = globalFlags.ZookeeperPreAllocSize
		req.ZookeeperMaxClientCnxns = globalFlags.ZookeeperMaxClientCnxns

		req.LogPrefix = globalFlags.LogPrefix
		req.DatabaseLogPath = globalFlags.DatabaseLogPath
		req.MonitorResultPath = globalFlags.MonitorResultPath
		req.GoogleCloudProjectName = globalFlags.GoogleCloudProjectName
		bts, err := ioutil.ReadFile(globalFlags.KeyPath)
		if err != nil {
			return err
		}
		req.StorageKey = string(bts)
		req.Bucket = globalFlags.Bucket
	}

	donec, errc := make(chan struct{}), make(chan error)
	for i := range peerIPs {
		go func(i int) {
			nreq := req

			nreq.ServerIndex = uint32(i)
			nreq.ZookeeperMyID = uint32(i + 1)
			ep := globalFlags.AgentEndpoints[nreq.ServerIndex]

			log.Printf("[%d] %s %s at %s\n", i, req.Operation, req.Database, ep)
			conn, err := grpc.Dial(ep, grpc.WithInsecure())
			if err != nil {
				log.Printf("[%d] error %v when connecting to %s\n", i, err, ep)
				errc <- err
				return
			}
			defer conn.Close()

			cli := agent.NewTransporterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Consul takes longer
			resp, err := cli.Transfer(ctx, &nreq)
			cancel()
			if err != nil {
				log.Printf("[%d] error %v when transferring to %s\n", i, err, ep)
				errc <- err
				return
			}
			log.Printf("[%d] Response from %s (%+v)\n", i, ep, resp)
			donec <- struct{}{}
		}(i)

		time.Sleep(time.Second)
	}
	cnt := 0
	for cnt != len(peerIPs) {
		select {
		case <-donec:
		case err := <-errc:
			return err
		}
		cnt++
	}
	return nil
}

func extractIPs(es []string) []string {
	var rs []string
	for _, v := range es {
		sl := strings.Split(v, ":")
		rs = append(rs, sl[0])
	}
	return rs
}
