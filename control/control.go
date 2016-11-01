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
		plog.Info("step 1: starting databases...")
		if err = step1(cfg); err != nil {
			return err
		}
	}

	if !cfg.Step2.Skip {
		println()
		time.Sleep(5 * time.Second)
		plog.Info("step 2: starting tests...")
		if err = step2(cfg); err != nil {
			return err
		}
	}

	if !cfg.Step3.Skip {
		println()
		time.Sleep(5 * time.Second)
		plog.Info("step 3: stopping databases...")
		if err = step3(cfg); err != nil {
			return err
		}
	}

	return nil
}

func step1(cfg Config) error { return bcastReq(cfg, agent.Request_Start) }

var (
	bar     *pb.ProgressBar
	results chan result
	wg      sync.WaitGroup
)

type values struct {
	bytes      [][]byte
	strings    []string
	sampleSize int
}

func newValues(cfg Config) (v values, rerr error) {
	if cfg.Step2.ValueTestDataPath == "" {
		v.bytes = [][]byte{randBytes(cfg.Step2.ValueSize)}
		v.strings = []string{string(v.bytes[0])}
		v.sampleSize = 1
		return
	}

	fs, err := walkDir(cfg.Step2.ValueTestDataPath)
	if err != nil {
		rerr = err
		return
	}
	for _, elem := range fs {
		bts, err := ioutil.ReadFile(elem.path)
		if err != nil {
			rerr = err
			return
		}
		v.bytes = append(v.bytes, bts)
		v.strings = append(v.strings, string(bts))
	}
	v.sampleSize = len(v.strings)
	return
}

func step2(cfg Config) error {
	vals, err := newValues(cfg)
	if err != nil {
		return err
	}
	switch cfg.Step2.BenchType {
	case "write":
		results = make(chan result)
		requests := make(chan request, cfg.Step2.Clients)
		bar = pb.New(cfg.Step2.TotalRequests)

		h, done := newWriteHandlers(cfg)
		if done != nil {
			defer done()
		}

		pdoneC := printReport(results, cfg)

		bar.Format("Bom !")
		bar.Start()
		for i := range h {
			wg.Add(1)
			go func(rh ReqHandler) {
				defer wg.Done()
				doHandler(context.Background(), rh, requests)
			}(h[i])
		}
		go generateWrites(cfg, vals, requests)
		wg.Wait()
		bar.Finish()

		close(results)
		<-pdoneC

		var totalKeysFunc func([]string) map[string]int64
		switch cfg.Database {
		case "etcdv2":
			totalKeysFunc = getTotalKeysEtcdv2
		case "etcdv3":
			totalKeysFunc = getTotalKeysEtcdv3
		case "zk", "zookeeper":
			totalKeysFunc = getTotalKeysZk
		case "consul":
			totalKeysFunc = getTotalKeysConsul
		}

		for k, v := range totalKeysFunc(cfg.DatabaseEndpoints) {
			plog.Infof("expected write total results [expected_total: %d | database: %q | endpoint: %q | number_of_keys: %d]", cfg.Step2.TotalRequests, cfg.Database, k, v)
		}

	case "read":
		key, value := sameKey(cfg.Step2.KeySize), vals.strings[0]

		switch cfg.Database {
		case "etcdv2":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, "etcdv2")
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Set(context.Background(), key, value, nil)
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, "etcdv2")
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, "etcdv2")
				os.Exit(1)
			}

		case "etcdv3":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, "etcdv3")
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
					totalConns:   1,
					totalClients: 1,
					// compressionTypeTxt: cfg.EtcdCompression,
				})
				_, err = clients[0].Do(context.Background(), clientv3.OpPut(key, value))
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, "etcdv3")
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, "etcdv3")
				os.Exit(1)
			}

		case "zk", "zookeeper":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, "zookeeper")
			var err error
			for i := 0; i < 7; i++ {
				conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = conns[0].Create("/"+key, vals.bytes[0], zkCreateFlags, zkCreateAcl)
				if err != nil {
					continue
				}
				for j := range conns {
					conns[j].Close()
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, "zookeeper")
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, "zookeeper")
				os.Exit(1)
			}

		case "consul":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, "consul")
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Put(&consulapi.KVPair{Key: key, Value: vals.bytes[0]}, nil)
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, "consul")
				break
			}
			if err != nil {
				plog.Errorf("write done [request: PUT | key: %q | database: %q]", key, "consul")
				os.Exit(1)
			}
		}

		results = make(chan result)
		requests := make(chan request, cfg.Step2.Clients)

		h, done := newReadHandlers(cfg)
		if done != nil {
			defer done()
		}

		bar = pb.New(cfg.Step2.TotalRequests)
		bar.Format("Bom !")
		bar.Start()

		for i := range h {
			wg.Add(1)
			go func(rh ReqHandler) {
				defer wg.Done()
				doHandler(context.Background(), rh, requests)
			}(h[i])
		}

		pdoneC := printReport(results, cfg)
		go generateReads(cfg, key, requests)
		wg.Wait()

		bar.Finish()

		close(results)
		<-pdoneC
	}

	return nil
}

func step3(cfg Config) error { return bcastReq(cfg, agent.Request_Stop) }

func bcastReq(cfg Config, op agent.Request_Operation) error {
	req := cfg.Request()
	req.Operation = op

	donec, errc := make(chan struct{}), make(chan error)
	for i := range cfg.PeerIPs {
		go func(i int) {
			if err := sendReq(cfg.AgentEndpoints[i], req, i); err != nil {
				errc <- err
			} else {
				donec <- struct{}{}
			}
		}(i)
		time.Sleep(time.Second)
	}

	var errs []error
	for cnt := 0; cnt != len(cfg.PeerIPs); cnt++ {
		select {
		case <-donec:
		case err := <-errc:
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func sendReq(ep string, req agent.Request, i int) error {
	req.ServerIndex = uint32(i)
	req.ZookeeperMyID = uint32(i + 1)

	plog.Infof("sending message [index: %d | operation: %q | database: %q | endpoint: %q]", i, req.Operation.String(), req.Database.String(), ep)

	conn, err := grpc.Dial(ep, grpc.WithInsecure())
	if err != nil {
		plog.Errorf("grpc.Dial connecting error (%v) [index: %d | endpoint: %q]", err, i, ep)
		return err
	}

	defer conn.Close()

	cli := agent.NewTransporterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Consul takes longer
	resp, err := cli.Transfer(ctx, &req)
	cancel()
	if err != nil {
		plog.Errorf("cli.Transfer error (%v) [index: %d | endpoint: %q]", err, i, ep)
		return err
	}

	plog.Infof("got response [index: %d | endpoint: %q | response: %+v]", i, ep, resp)
	return nil
}

func newReadHandlers(cfg Config) (rhs []ReqHandler, done func()) {
	rhs = make([]ReqHandler, cfg.Step2.Clients)

	switch cfg.Database {
	case "etcdv2":
		conns := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newGetEtcd2(conns[i])
		}
	case "etcdv3":
		clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
			totalConns:   cfg.Step2.Connections,
			totalClients: cfg.Step2.Clients,
			// compressionTypeTxt: cfg.EtcdCompression,
		})
		for i := range clients {
			rhs[i] = newGetEtcd3(clients[i].KV)
		}
		done = func() {
			for i := range clients {
				clients[i].Close()
			}
		}
	case "zk", "zookeeper":
		conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newGetZK(conns[i])
		}
		done = func() {
			for i := range conns {
				conns[i].Close()
			}
		}
	case "consul":
		conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newGetConsul(conns[i])
		}
	}
	return
}

func newWriteHandlers(cfg Config) (rhs []ReqHandler, done func()) {
	rhs = make([]ReqHandler, cfg.Step2.Clients)
	switch cfg.Database {
	case "etcdv2":
		conns := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newPutEtcd2(conns[i])
		}
	case "etcdv3":
		etcdClients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
			totalConns:   cfg.Step2.Connections,
			totalClients: cfg.Step2.Clients,
			// compressionTypeTxt: cfg.EtcdCompression,
		})
		for i := range etcdClients {
			rhs[i] = newPutEtcd3(etcdClients[i])
		}
		done = func() {
			for i := range etcdClients {
				etcdClients[i].Close()
			}
		}
	case "zk", "zookeeper":
		if cfg.Step2.SameKey {
			key := sameKey(cfg.Step2.KeySize)
			valueBts := randBytes(cfg.Step2.ValueSize)
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, "zookeeper")
			var err error
			for i := 0; i < 7; i++ {
				conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = conns[0].Create("/"+key, valueBts, zkCreateFlags, zkCreateAcl)
				if err != nil {
					continue
				}
				for j := range conns {
					conns[j].Close()
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, "zookeeper")
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, "zookeeper")
				os.Exit(1)
			}
		}

		conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			if cfg.Step2.SameKey {
				rhs[i] = newPutOverwriteZK(conns[i])
			} else {
				rhs[i] = newPutCreateZK(conns[i])
			}
		}
		done = func() {
			for i := range conns {
				conns[i].Close()
			}
		}
	case "consul":
		conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newPutConsul(conns[i])
		}
	}
	return
}

func generateReads(cfg Config, key string, requests chan request) {
	defer close(requests)

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
		if cfg.Step2.RequestIntervalMs > 0 {
			time.Sleep(time.Duration(cfg.Step2.RequestIntervalMs) * time.Millisecond)
		}
	}
}

func generateWrites(cfg Config, vals values, requests chan request) {
	defer close(requests)

	for i := 0; i < cfg.Step2.TotalRequests; i++ {
		if cfg.Database == "etcdv3" && cfg.Step2.Etcdv3CompactionCycle > 0 && i%cfg.Step2.Etcdv3CompactionCycle == 0 {
			plog.Infof("starting compaction [index: %d | database: %q]", i, "etcdv3")
			wg.Add(1)
			go func() {
				defer wg.Done()
				ccfg := etcdv3ClientCfg{1, 1}
				c := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, ccfg)
				compactKV(c)
			}()
		}

		k := sequentialKey(cfg.Step2.KeySize, i)
		if cfg.Step2.SameKey {
			k = sameKey(cfg.Step2.KeySize)
		}

		v := vals.bytes[i%vals.sampleSize]
		vs := vals.strings[i%vals.sampleSize]

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
		if cfg.Step2.RequestIntervalMs > 0 {
			time.Sleep(time.Duration(cfg.Step2.RequestIntervalMs) * time.Millisecond)
		}
	}
}
