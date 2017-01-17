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

package control

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/coreos/dbtester/agent/agentpb"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/report"
	"github.com/gyuho/dataframe"
	consulapi "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

type values struct {
	bytes      [][]byte
	strings    []string
	sampleSize int
}

func newValues(cfg Config) (v values, rerr error) {
	v.bytes = [][]byte{randBytes(cfg.Step2.ValueSize)}
	v.strings = []string{string(v.bytes[0])}
	v.sampleSize = 1
	return
}

type benchmark struct {
	cfg Config

	bar        *pb.ProgressBar
	report     report.Report
	reportDone <-chan report.Stats
	stats      report.Stats

	reqHandlers []ReqHandler
	reqGen      func(chan<- request)
	reqDone     func()
	wg          sync.WaitGroup

	mu           sync.RWMutex
	inflightReqs chan request
}

func newBenchmark(cfg Config, reqHandlers []ReqHandler, reqDone func(), reqGen func(chan<- request)) (b *benchmark) {
	b = &benchmark{
		cfg:         cfg,
		bar:         pb.New(cfg.Step2.TotalRequests),
		reqHandlers: reqHandlers,
		reqGen:      reqGen,
		reqDone:     reqDone,
		wg:          sync.WaitGroup{},
	}
	b.inflightReqs = make(chan request, b.cfg.Step2.Clients)

	b.bar.Format("Bom !")
	b.bar.Start()
	b.report = report.NewReport("%4.4f")
	return
}

func (b *benchmark) reset(clientsN int, reqHandlers []ReqHandler, reqDone func(), reqGen func(chan<- request)) {
	b.reqHandlers = reqHandlers
	b.reqDone = reqDone
	b.reqGen = reqGen

	// inflight requests will be dropped!
	b.mu.Lock()
	b.inflightReqs = make(chan request, clientsN)
	b.mu.Unlock()
}

func (b *benchmark) getInflightsReqs() (ch chan request) {
	b.mu.RLock()
	ch = b.inflightReqs
	b.mu.RUnlock()
	return
}

func (b *benchmark) startRequests() {
	for i := range b.reqHandlers {
		b.wg.Add(1)
		go func(rh ReqHandler) {
			defer b.wg.Done()
			for req := range b.getInflightsReqs() {
				st := time.Now()
				err := rh(context.Background(), &req)
				b.report.Results() <- report.Result{Err: err, Start: st, End: time.Now()}
				b.bar.Increment()
			}
		}(b.reqHandlers[i])
	}
	go b.reqGen(b.getInflightsReqs())
	b.reportDone = b.report.Stats()
}

func (b *benchmark) waitRequestsEnd() {
	b.wg.Wait()
	if b.reqDone != nil {
		b.reqDone() // cancel connections
	}
}

func (b *benchmark) finishReports() {
	close(b.report.Results())
	b.bar.Finish()
	st := <-b.reportDone
	b.stats = st
}

func (b *benchmark) waitAll() {
	b.waitRequestsEnd()
	b.finishReports()
}

func printStats(st report.Stats) {
	// to be piped to cfg.Log via stdout when dbtester executed
	if len(st.Lats) > 0 {
		fmt.Printf("Total: %v\n", st.Total)
		fmt.Printf("Slowest: %f secs\n", st.Slowest)
		fmt.Printf("Fastest: %f secs\n", st.Fastest)
		fmt.Printf("Average: %f secs\n", st.Average)
		fmt.Printf("Requests/sec: %4.4f\n", st.RPS)
	}
}

func saveDataLatencyDistributionSummary(cfg Config, st report.Stats) {
	fr := dataframe.New()

	c1 := dataframe.NewColumn("TOTAL-SECONDS")
	c1.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", st.Total.Seconds())))
	if err := fr.AddColumn(c1); err != nil {
		plog.Fatal(err)
	}

	c2 := dataframe.NewColumn("SLOWEST-LATENCY-MS")
	c2.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", 1000*st.Slowest)))
	if err := fr.AddColumn(c2); err != nil {
		plog.Fatal(err)
	}

	c3 := dataframe.NewColumn("FASTEST-LATENCY-MS")
	c3.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", 1000*st.Fastest)))
	if err := fr.AddColumn(c3); err != nil {
		plog.Fatal(err)
	}

	c4 := dataframe.NewColumn("AVERAGE-LATENCY-MS")
	c4.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", 1000*st.Average)))
	if err := fr.AddColumn(c4); err != nil {
		plog.Fatal(err)
	}

	c5 := dataframe.NewColumn("STDDEV-LATENCY-MS")
	c5.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", 1000*st.Stddev)))
	if err := fr.AddColumn(c5); err != nil {
		plog.Fatal(err)
	}

	c6 := dataframe.NewColumn("REQUESTS-PER-SECOND")
	c6.PushBack(dataframe.NewStringValue(fmt.Sprintf("%4.4f", 1000*st.RPS)))
	if err := fr.AddColumn(c6); err != nil {
		plog.Fatal(err)
	}

	for errName, errN := range st.ErrorDist {
		errcol := dataframe.NewColumn(fmt.Sprintf("ERROR: %q", errName))
		errcol.PushBack(dataframe.NewStringValue(errN))
		if err := fr.AddColumn(errcol); err != nil {
			plog.Fatal(err)
		}
	}

	if err := fr.CSVHorizontal(cfg.DataLatencyDistributionSummary); err != nil {
		plog.Fatal(err)
	}
}

func saveDataLatencyDistributionPercentile(cfg Config, st report.Stats) {
	pctls, seconds := report.Percentiles(st.Lats)
	c1 := dataframe.NewColumn("LATENCY-PERCENTILE")
	c2 := dataframe.NewColumn("LATENCY-MS")
	for i := range pctls {
		c1.PushBack(dataframe.NewStringValue(fmt.Sprintf("p-%f", pctls[i])))
		c2.PushBack(dataframe.NewStringValue(fmt.Sprintf("%f", 1000*seconds[i])))
	}

	fr := dataframe.New()
	if err := fr.AddColumn(c1); err != nil {
		plog.Fatal(err)
	}
	if err := fr.AddColumn(c2); err != nil {
		plog.Fatal(err)
	}
	if err := fr.CSVHorizontal(cfg.DataLatencyDistributionPercentile); err != nil {
		plog.Fatal(err)
	}
}

func saveDataLatencyDistributionAll(cfg Config, st report.Stats) {
	min := int64(math.MaxInt64)
	max := int64(-100000)
	rm := make(map[int64]int64)
	for _, lt := range st.Lats {
		// convert second(float64) to millisecond
		ms := lt * 1000

		// truncate all digits below 10ms
		// (e.g. 125.11ms becomes 120ms)
		v := int64(math.Trunc(ms/10) * 10)
		if _, ok := rm[v]; !ok {
			rm[v] = 1
		} else {
			rm[v]++
		}

		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}

	c1 := dataframe.NewColumn("LATENCY-MS")
	c2 := dataframe.NewColumn("COUNT")
	cur := min
	for {
		c1.PushBack(dataframe.NewStringValue(fmt.Sprintf("%d", int64(cur))))
		v, ok := rm[cur]
		if ok {
			c2.PushBack(dataframe.NewStringValue(fmt.Sprintf("%d", v)))
		} else {
			c2.PushBack(dataframe.NewStringValue("0"))
		}
		cur += 10
		if cur-10 == max { // was last point
			break
		}
	}
	fr := dataframe.New()
	if err := fr.AddColumn(c1); err != nil {
		plog.Fatal(err)
	}
	if err := fr.AddColumn(c2); err != nil {
		plog.Fatal(err)
	}
	if err := fr.CSV(cfg.DataLatencyDistributionAll); err != nil {
		plog.Fatal(err)
	}
}

func saveDataLatencyThroughputTimeseries(cfg Config, st report.Stats) {
	c1 := dataframe.NewColumn("UNIX-TS")
	c2 := dataframe.NewColumn("AVG-LATENCY-MS")
	c3 := dataframe.NewColumn("AVG-THROUGHPUT")
	for i := range st.TimeSeries {
		c1.PushBack(dataframe.NewStringValue(fmt.Sprintf("%d", st.TimeSeries[i].Timestamp)))
		c2.PushBack(dataframe.NewStringValue(fmt.Sprintf("%f", toMillisecond(st.TimeSeries[i].AvgLatency))))
		c3.PushBack(dataframe.NewStringValue(fmt.Sprintf("%d", st.TimeSeries[i].ThroughPut)))
	}

	fr := dataframe.New()
	if err := fr.AddColumn(c1); err != nil {
		plog.Fatal(err)
	}
	if err := fr.AddColumn(c2); err != nil {
		plog.Fatal(err)
	}
	if err := fr.AddColumn(c3); err != nil {
		plog.Fatal(err)
	}
	if err := fr.CSV(cfg.DataLatencyThroughputTimeseries); err != nil {
		plog.Fatal(err)
	}
}

func generateReport(cfg Config, h []ReqHandler, reqDone func(), reqGen func(chan<- request)) {
	b := newBenchmark(cfg, h, reqDone, reqGen)
	b.startRequests()
	b.waitAll()

	printStats(b.stats)

	// cfg.DataLatencyDistributionSummary
	saveDataLatencyDistributionSummary(cfg, b.stats)

	// cfg.DataLatencyDistributionPercentile
	saveDataLatencyDistributionPercentile(cfg, b.stats)

	// cfg.DataLatencyDistributionAll
	saveDataLatencyDistributionAll(cfg, b.stats)

	// cfg.DataLatencyThroughputTimeseries
	saveDataLatencyThroughputTimeseries(cfg, b.stats)
}

func step2StressDatabase(cfg Config) error {
	vals, err := newValues(cfg)
	if err != nil {
		return err
	}

	switch cfg.Step2.BenchType {
	case "write":
		plog.Println("write generateReport is started...")
		if cfg.Step2.ConnectionsClientsMax == 0 {
			h, done := newWriteHandlers(cfg)
			reqGen := func(inflightReqs chan<- request) { generateWrites(cfg, vals, inflightReqs) }
			generateReport(cfg, h, done, reqGen)
		} else {
			// need client number increase
			// TODO: currently, request range is 100000 (fixed)
			// e.g. 2M requests, starts with clients 100, range 100K
			// at 2M requests point, there will be 2K clients (20 * 100)
			copied := cfg
			copied.Step2.TotalRequests = 100000
			h, done := newWriteHandlers(copied)
			reqGen := func(inflightReqs chan<- request) { generateWrites(copied, vals, inflightReqs) }
			b := newBenchmark(copied, h, done, reqGen)

			reqCompleted := 0
			for reqCompleted >= cfg.Step2.TotalRequests {
				plog.Infof("signaling agent on client number %d", copied.Step2.Clients)
				// signal agent on the client number
				if err := bcastReq(copied, agentpb.Request_Heartbeat); err != nil {
					return err
				}

				// generate request
				b.startRequests()

				// wait until 100000 requests are finished
				// do not finish reports yet
				b.waitRequestsEnd()

				// update request handlers, generator
				copied.Step2.Clients += copied.Step2.ConnectionsClientsDelta
				if copied.Step2.Clients > copied.Step2.ConnectionsClientsMax {
					copied.Step2.Clients = copied.Step2.ConnectionsClientsMax
				}
				h, done = newWriteHandlers(copied)
				reqGen = func(inflightReqs chan<- request) { generateWrites(copied, vals, inflightReqs) }
				b.reset(copied.Step2.Clients, h, done, reqGen)
				plog.Infof("updated client number %d", copied.Step2.Clients)

				// after one range of requests are finished
				reqCompleted += 100000
			}

			// finish reports
			b.finishReports()
		}
		plog.Println("write generateReport is finished...")

		plog.Println("checking total keys on", cfg.DatabaseEndpoints)
		var totalKeysFunc func([]string) map[string]int64
		switch cfg.Database {
		case "etcdv2":
			totalKeysFunc = getTotalKeysEtcdv2
		case "etcdv3":
			totalKeysFunc = getTotalKeysEtcdv3
		case "zookeeper", "zetcd":
			totalKeysFunc = getTotalKeysZk
		case "consul", "cetcd":
			totalKeysFunc = getTotalKeysConsul
		}
		for k, v := range totalKeysFunc(cfg.DatabaseEndpoints) {
			plog.Infof("expected write total results [expected_total: %d | database: %q | endpoint: %q | number_of_keys: %d]",
				cfg.Step2.TotalRequests, cfg.Database, k, v)
		}

	case "read":
		key, value := sameKey(cfg.Step2.KeySize), vals.strings[0]

		switch cfg.Database {
		case "etcdv2":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, cfg.Database)
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Set(context.Background(), key, value, nil)
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, cfg.Database)
				os.Exit(1)
			}

		case "etcdv3":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, cfg.Database)
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
					totalConns:   1,
					totalClients: 1,
				})
				_, err = clients[0].Do(context.Background(), clientv3.OpPut(key, value))
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, cfg.Database)
				os.Exit(1)
			}

		case "zookeeper", "zetcd":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, cfg.Database)
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
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, cfg.Database)
				os.Exit(1)
			}

		case "consul", "cetcd":
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, cfg.Database)
			var err error
			for i := 0; i < 7; i++ {
				clients := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				_, err = clients[0].Put(&consulapi.KVPair{Key: key, Value: vals.bytes[0]}, nil)
				if err != nil {
					continue
				}
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				break
			}
			if err != nil {
				plog.Errorf("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				os.Exit(1)
			}
		}

		h, done := newReadHandlers(cfg)
		reqGen := func(inflightReqs chan<- request) { generateReads(cfg, key, inflightReqs) }
		generateReport(cfg, h, done, reqGen)
		plog.Println("read generateReport is finished...")

	case "read-oneshot":
		key, value := sameKey(cfg.Step2.KeySize), vals.strings[0]
		plog.Infof("writing key for read-oneshot [key: %q | database: %q]", key, cfg.Database)
		var err error
		switch cfg.Database {
		case "etcdv2":
			clients := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, 1)
			_, err = clients[0].Set(context.Background(), key, value, nil)

		case "etcdv3":
			clients := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
				totalConns:   1,
				totalClients: 1,
			})
			_, err = clients[0].Do(context.Background(), clientv3.OpPut(key, value))
			clients[0].Close()

		case "zookeeper", "zetcd":
			conns := mustCreateConnsZk(cfg.DatabaseEndpoints, 1)
			_, err = conns[0].Create("/"+key, vals.bytes[0], zkCreateFlags, zkCreateAcl)
			conns[0].Close()

		case "consul", "cetcd":
			clients := mustCreateConnsConsul(cfg.DatabaseEndpoints, 1)
			_, err = clients[0].Put(&consulapi.KVPair{Key: key, Value: vals.bytes[0]}, nil)
		}
		if err != nil {
			plog.Errorf("write error on read-oneshot (%v)", err)
			os.Exit(1)
		}

		h := newReadOneshotHandlers(cfg)
		reqGen := func(inflightReqs chan<- request) { generateReads(cfg, key, inflightReqs) }
		generateReport(cfg, h, nil, reqGen)
		plog.Println("read-oneshot generateReport is finished...")
	}

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
		})
		for i := range clients {
			rhs[i] = newGetEtcd3(clients[i].KV)
		}
		done = func() {
			for i := range clients {
				clients[i].Close()
			}
		}
	case "zookeeper", "zetcd":
		conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newGetZK(conns[i])
		}
		done = func() {
			for i := range conns {
				conns[i].Close()
			}
		}
	case "consul", "cetcd":
		conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newGetConsul(conns[i])
		}
	}
	return rhs, done
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
		})
		for i := range etcdClients {
			rhs[i] = newPutEtcd3(etcdClients[i])
		}
		done = func() {
			for i := range etcdClients {
				etcdClients[i].Close()
			}
		}
	case "zookeeper", "zetcd":
		if cfg.Step2.SameKey {
			key := sameKey(cfg.Step2.KeySize)
			valueBts := randBytes(cfg.Step2.ValueSize)
			plog.Infof("write started [request: PUT | key: %q | database: %q]", key, cfg.Database)
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
				plog.Infof("write done [request: PUT | key: %q | database: %q]", key, cfg.Database)
				break
			}
			if err != nil {
				plog.Errorf("write error [request: PUT | key: %q | database: %q]", key, cfg.Database)
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
	case "consul", "cetcd":
		conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, cfg.Step2.Connections)
		for i := range conns {
			rhs[i] = newPutConsul(conns[i])
		}
	}
	return
}

func newReadOneshotHandlers(cfg Config) []ReqHandler {
	rhs := make([]ReqHandler, cfg.Step2.Clients)
	switch cfg.Database {
	case "etcdv2":
		for i := range rhs {
			rhs[i] = func(ctx context.Context, req *request) error {
				conns := mustCreateClientsEtcdv2(cfg.DatabaseEndpoints, 1)
				return newGetEtcd2(conns[0])(ctx, req)
			}
		}
	case "etcdv3":
		for i := range rhs {
			rhs[i] = func(ctx context.Context, req *request) error {
				conns := mustCreateClientsEtcdv3(cfg.DatabaseEndpoints, etcdv3ClientCfg{
					totalConns:   1,
					totalClients: 1,
				})
				defer conns[0].Close()
				return newGetEtcd3(conns[0])(ctx, req)
			}
		}
	case "zookeeper", "zetcd":
		for i := range rhs {
			rhs[i] = func(ctx context.Context, req *request) error {
				conns := mustCreateConnsZk(cfg.DatabaseEndpoints, cfg.Step2.Connections)
				defer conns[0].Close()
				return newGetZK(conns[0])(ctx, req)
			}
		}
	case "consul", "cetcd":
		for i := range rhs {
			rhs[i] = func(ctx context.Context, req *request) error {
				conns := mustCreateConnsConsul(cfg.DatabaseEndpoints, 1)
				return newGetConsul(conns[0])(ctx, req)
			}
		}
	}
	return rhs
}

func generateReads(cfg Config, key string, inflightReqs chan<- request) {
	defer close(inflightReqs)

	var rateLimiter *rate.Limiter
	if cfg.Step2.RequestsPerSecond > 0 {
		rateLimiter = rate.NewLimiter(rate.Limit(cfg.Step2.RequestsPerSecond), cfg.Step2.RequestsPerSecond)
	}

	for i := 0; i < cfg.Step2.TotalRequests; i++ {
		if rateLimiter != nil {
			rateLimiter.Wait(context.TODO())
		}

		switch cfg.Database {
		case "etcdv2":
			// serializable read by default
			inflightReqs <- request{etcdv2Op: etcdv2Op{key: key}}

		case "etcdv3":
			opts := []clientv3.OpOption{clientv3.WithRange("")}
			if cfg.Step2.StaleRead {
				opts = append(opts, clientv3.WithSerializable())
			}
			inflightReqs <- request{etcdv3Op: clientv3.OpGet(key, opts...)}

		case "zookeeper", "zetcd":
			op := zkOp{key: key}
			if cfg.Step2.StaleRead {
				op.staleRead = true
			}
			inflightReqs <- request{zkOp: op}

		case "consul", "cetcd":
			op := consulOp{key: key}
			if cfg.Step2.StaleRead {
				op.staleRead = true
			}
			inflightReqs <- request{consulOp: op}
		}
	}
}

func generateWrites(cfg Config, vals values, inflightReqs chan<- request) {
	var rateLimiter *rate.Limiter
	if cfg.Step2.RequestsPerSecond > 0 {
		rateLimiter = rate.NewLimiter(rate.Limit(cfg.Step2.RequestsPerSecond), cfg.Step2.RequestsPerSecond)
	}

	var wg sync.WaitGroup
	defer func() {
		close(inflightReqs)
		wg.Wait()
	}()

	for i := 0; i < cfg.Step2.TotalRequests; i++ {
		k := sequentialKey(cfg.Step2.KeySize, i)
		if cfg.Step2.SameKey {
			k = sameKey(cfg.Step2.KeySize)
		}

		v := vals.bytes[i%vals.sampleSize]
		vs := vals.strings[i%vals.sampleSize]

		if rateLimiter != nil {
			rateLimiter.Wait(context.TODO())
		}

		switch cfg.Database {
		case "etcdv2":
			inflightReqs <- request{etcdv2Op: etcdv2Op{key: k, value: vs}}
		case "etcdv3":
			inflightReqs <- request{etcdv3Op: clientv3.OpPut(k, vs)}
		case "zookeeper", "zetcd":
			inflightReqs <- request{zkOp: zkOp{key: "/" + k, value: v}}
		case "consul", "cetcd":
			inflightReqs <- request{consulOp: consulOp{key: k, value: v}}
		}
	}
}
