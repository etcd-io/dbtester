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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/dbtester/agent"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/cheggaaa/pb"
	"github.com/coreos/etcd/clientv3"
	consulapi "github.com/hashicorp/consul/api"
)

var (
	Command = &cobra.Command{
		Use:   "control",
		Short: "Controls tests.",
		RunE:  CommandFunc,
	}
	configPath string
)

func init() {
	Command.PersistentFlags().StringVarP(&configPath, "config", "c", "", "YAML configuration file path.")
}

func CommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := ReadConfig(configPath)
	if err != nil {
		return err
	}
	switch cfg.Database {
	case "etcdv2":
	case "etcdv3":
	case "zk", "zookeeper":
	case "consul":
	default:
		return fmt.Errorf("%q is not supported", cfg.Database)
	}
	if !cfg.Step2.Skip {
		switch cfg.Step2.BenchType {
		case "write":
		case "read":
		default:
			return fmt.Errorf("%q is not supported", cfg.Step2.BenchType)
		}
	}

	bts, err := ioutil.ReadFile(cfg.GoogleCloudStorageKeyPath)
	if err != nil {
		return err
	}
	cfg.GoogleCloudStorageKey = string(bts)

	cfg.PeerIPString = strings.Join(cfg.PeerIPs, "___") // protoc sorts the 'repeated' type data
	cfg.AgentEndpoints = make([]string, len(cfg.PeerIPs))
	cfg.DatabaseEndpoints = make([]string, len(cfg.PeerIPs))
	for i := range cfg.PeerIPs {
		cfg.AgentEndpoints[i] = fmt.Sprintf("%s:%d", cfg.PeerIPs[i], cfg.AgentPort)
	}
	for i := range cfg.PeerIPs {
		cfg.DatabaseEndpoints[i] = fmt.Sprintf("%s:%d", cfg.PeerIPs[i], cfg.DatabasePort)
	}

	println()
	if !cfg.Step1.Skip {
		log.Println("Step 1: starting databases...")

		if err = step1(cfg); err != nil {
			return err
		}
	}

	if !cfg.Step2.Skip {
		println()
		time.Sleep(5 * time.Second)
		log.Println("Step 2: starting tests...")

		if err = step2(cfg); err != nil {
			return err
		}
	}

	if !cfg.Step3.Skip {
		println()
		time.Sleep(5 * time.Second)
		log.Println("Step 3: stopping databases...")

		if err = step3(cfg); err != nil {
			return err
		}
	}

	return nil
}

func step1(cfg Config) error {
	req := agent.Request{}

	req.Operation = agent.Request_Start
	req.TestName = cfg.TestName
	req.GoogleCloudProjectName = cfg.GoogleCloudProjectName
	req.GoogleCloudStorageKey = cfg.GoogleCloudStorageKey
	req.GoogleCloudStorageBucketName = cfg.GoogleCloudStorageBucketName

	switch cfg.Database {
	case "etcdv2":
		req.Database = agent.Request_etcdv2

	case "etcdv3":
		req.Database = agent.Request_etcdv3

	case "zk", "zookeeper":
		cfg.Database = "zookeeper"
		req.Database = agent.Request_ZooKeeper

	case "consul":
		req.Database = agent.Request_Consul
	}

	req.DatabaseLogPath = cfg.Step1.DatabaseLogPath
	req.MonitorLogPath = cfg.Step1.MonitorLogPath
	req.PeerIPString = cfg.PeerIPString

	req.ZookeeperMaxClientCnxns = cfg.Step1.ZookeeperMaxClientCnxns
	req.ZookeeperSnapCount = cfg.Step1.ZookeeperSnapCount

	donec, errc := make(chan struct{}), make(chan error)
	for i := range cfg.PeerIPs {

		go func(i int) {
			nreq := req

			nreq.ServerIndex = uint32(i)
			nreq.ZookeeperMyID = uint32(i + 1)
			ep := cfg.AgentEndpoints[nreq.ServerIndex]

			log.Printf("[%d] %s %s at %s", i, req.Operation, req.Database, ep)

			conn, err := grpc.Dial(ep, grpc.WithInsecure())
			if err != nil {
				log.Printf("[%d] error %v when connecting to %s", i, err, ep)
				errc <- err
				return
			}

			defer conn.Close()

			cli := agent.NewTransporterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Consul takes longer
			resp, err := cli.Transfer(ctx, &nreq)
			cancel()
			if err != nil {
				log.Printf("[%d] error %v when transferring to %s", i, err, ep)
				errc <- err
				return
			}

			log.Printf("[%d] Response from %s (%+v)", i, ep, resp)
			donec <- struct{}{}
		}(i)

		time.Sleep(time.Second)
	}

	cnt := 0
	for cnt != len(cfg.PeerIPs) {
		select {
		case <-donec:
		case err := <-errc:
			return err
		}
		cnt++
	}

	return nil
}

var (
	bar     *pb.ProgressBar
	results chan result
	wg      sync.WaitGroup
)

func step2(cfg Config) error {
	switch cfg.Step2.BenchType {
	case "write":

		results = make(chan result)
		requests := make(chan request, cfg.Step2.Clients)
		bar = pb.New(cfg.Step2.TotalRequests)

		v := randBytes(cfg.Step2.ValueSize)
		vs := string(v)

		bar.Format("Bom !")
		bar.Start()

		var etcdClients []*clientv3.Client
		switch cfg.Database {
		case "etcdv2":
			conns := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			for i := range conns {
				wg.Add(1)
				go doPutEtcdv2(context.Background(), conns[i], requests)
			}

		case "etcdv3":
			etcdClients = mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, cfg.Step2.Clients, cfg.Step2.Connections)
			for i := range etcdClients {
				wg.Add(1)
				go doPutEtcdv3(context.Background(), etcdClients[i], requests)
			}
			defer func() {
				for i := range etcdClients {
					etcdClients[i].Close()
				}
			}()

		case "zk", "zookeeper":
			conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			defer func() {
				for i := range conns {
					conns[i].Close()
				}
			}()
			for i := range conns {
				wg.Add(1)
				go doPutZk(conns[i], requests)
			}

		case "consul":
			conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			for i := range conns {
				wg.Add(1)
				go doPutConsul(conns[i], requests)
			}
		}

		pdoneC := printReport(results, cfg)
		go func() {
			for i := 0; i < cfg.Step2.TotalRequests; i++ {
				if cfg.Database == "etcdv3" && cfg.Step2.Etcdv3CompactionCycle > 0 && i%cfg.Step2.Etcdv3CompactionCycle == 0 {
					log.Printf("etcdv3 starting compaction at %d put request", i)
					go func() {
						compactKV(etcdClients)
					}()
				}

				k := sequentialKey(cfg.Step2.KeySize, i)

				switch cfg.Database {
				case "etcdv2":
					requests <- request{etcdv2Op: etcdv2Op{key: k, value: vs}}

				case "etcdv3":
					requests <- request{etcdv3Op: clientv3.OpPut(k, vs)}

				case "zk", "zookeeper":
					requests <- request{zkOp: zkOp{key: "/" + k, value: v}}

				case "consul":
					requests <- request{consulOp: consulOp{key: k, value: v}}
				}
			}
			close(requests)
		}()

		wg.Wait()

		bar.Finish()

		close(results)
		<-pdoneC

		log.Println("Expected Write Total:", cfg.Step2.TotalRequests)
		switch cfg.Database {
		case "etcdv2":
			for k, v := range getTotalKeysEtcdv2(cfg.DatabaseEndpoints) {
				fmt.Println("Endpoint      :", k)
				fmt.Println("Number of Keys:", v)
				fmt.Println()
			}

		case "etcdv3":
			for k, v := range getTotalKeysEtcdv3(cfg.DatabaseEndpoints) {
				fmt.Println("Endpoint      :", k)
				fmt.Println("Number of Keys:", v)
				fmt.Println()
			}

		case "zk", "zookeeper":
			for k, v := range getTotalKeysZk(cfg.DatabaseEndpoints) {
				fmt.Println("Endpoint      :", k)
				fmt.Println("Number of Keys:", v)
				fmt.Println()
			}

		case "consul":
			for k, v := range getTotalKeysConsul(cfg.DatabaseEndpoints) {
				fmt.Println("Endpoint      :", k)
				fmt.Println("Number of Keys:", v)
				fmt.Println()
			}
		}

	case "read":
		var (
			key      = string(randBytes(cfg.Step2.KeySize))
			valueBts = randBytes(cfg.Step2.ValueSize)
			value    = string(valueBts)
		)
		switch cfg.Database {
		case "etcdv2":
			log.Printf("PUT '%s' to etcdv2", key)
			var err error
			for i := 0; i < 5; i++ {
				clients := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Set(context.Background(), key, value, nil)
				if err != nil {
					continue
				}
				log.Printf("Done with PUT '%s' to etcdv2", key)
				break
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		case "etcdv3":
			log.Printf("PUT '%s' to etcd", key)
			var err error
			for i := 0; i < 5; i++ {
				clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, 1, 1)
				_, err = clients[0].Do(context.Background(), clientv3.OpPut(key, value))
				if err != nil {
					continue
				}
				log.Printf("Done with PUT '%s' to etcd", key)
				break
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		case "zk", "zookeeper":
			log.Printf("PUT '/%s' to Zookeeper", key)
			var err error
			for i := 0; i < 5; i++ {
				conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = conns[0].Create("/"+key, valueBts, zkCreateFlags, zkCreateAcl)
				if err != nil {
					continue
				}
				log.Printf("Done with PUT '/%s' to Zookeeper", key)
				break
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		case "consul":
			log.Printf("PUT '%s' to Consul", key)
			var err error
			for i := 0; i < 5; i++ {
				clients := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Put(&consulapi.KVPair{Key: key, Value: valueBts}, nil)
				if err != nil {
					continue
				}
				log.Printf("Done with PUT '%s' to Consul", key)
				break
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		results = make(chan result)
		requests := make(chan request, cfg.Step2.Clients)
		bar = pb.New(cfg.Step2.TotalRequests)

		bar.Format("Bom !")
		bar.Start()

		switch cfg.Database {
		case "etcdv2":
			conns := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			for i := range conns {
				wg.Add(1)
				go doRangeEtcdv2(conns[i], requests)
			}

		case "etcdv3":
			clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, cfg.Step2.Clients, cfg.Step2.Connections)
			for i := range clients {
				wg.Add(1)
				go doRangeEtcdv3(clients[i].KV, requests)
			}
			defer func() {
				for i := range clients {
					clients[i].Close()
				}
			}()

		case "zk", "zookeeper":
			conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			defer func() {
				for i := range conns {
					conns[i].Close()
				}
			}()
			for i := range conns {
				wg.Add(1)
				go doRangeZk(conns[i], requests)
			}

		case "consul":
			conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
			for i := range conns {
				wg.Add(1)
				go doRangeConsul(conns[i], requests)
			}
		}

		pdoneC := printReport(results, cfg)
		go func() {
			for i := 0; i < cfg.Step2.TotalRequests; i++ {
				switch cfg.Database {
				case "etcdv2":
					// serializable read by default
					requests <- request{etcdv2Op: etcdv2Op{key: key}}

				case "etcdv3":
					opts := []clientv3.OpOption{clientv3.WithRange("")}
					if cfg.Step2.LocalRead {
						opts = append(opts, clientv3.WithSerializable())
					}
					requests <- request{etcdv3Op: clientv3.OpGet(key, opts...)}

				case "zk", "zookeeper":
					// serializable read by default
					requests <- request{zkOp: zkOp{key: key}}

				case "consul":
					// serializable read by default
					requests <- request{consulOp: consulOp{key: key}}
				}
			}
			close(requests)
		}()

		wg.Wait()

		bar.Finish()

		close(results)
		<-pdoneC
	}

	return nil
}

func step3(cfg Config) error {
	req := agent.Request{}
	req.Operation = agent.Request_Stop

	switch cfg.Database {
	case "etcdv2":
		req.Database = agent.Request_etcdv2

	case "etcdv3":
		req.Database = agent.Request_etcdv3

	case "zk":
		cfg.Database = "zookeeper"
		req.Database = agent.Request_ZooKeeper

	case "zookeeper":
		req.Database = agent.Request_ZooKeeper

	case "consul":
		req.Database = agent.Request_Consul
	}

	donec, errc := make(chan struct{}), make(chan error)
	for i := range cfg.PeerIPs {

		go func(i int) {
			ep := cfg.AgentEndpoints[i]
			log.Printf("[%d] %s %s at %s", i, req.Operation, req.Database, ep)

			conn, err := grpc.Dial(ep, grpc.WithInsecure())
			if err != nil {
				log.Printf("[%d] error %v when connecting to %s", i, err, ep)
				errc <- err
				return
			}

			defer conn.Close()

			cli := agent.NewTransporterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Consul takes longer
			resp, err := cli.Transfer(ctx, &req)
			cancel()
			if err != nil {
				log.Printf("[%d] error %v when transferring to %s", i, err, ep)
				errc <- err
				return
			}

			log.Printf("[%d] Response from %s (%+v)", i, ep, resp)
			donec <- struct{}{}
		}(i)

		time.Sleep(time.Second)
	}

	cnt := 0
	for cnt != len(cfg.PeerIPs) {
		select {
		case <-donec:
		case err := <-errc:
			return err
		}
		cnt++
	}

	return nil
}
