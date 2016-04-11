// Copyright 2015 CoreOS, Inc.
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
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	mrand "math/rand"

	clientv2 "github.com/coreos/etcd/client"
	"github.com/coreos/etcd/clientv3"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	zkCreateFlags = int32(0)
	zkCreateAcl   = zk.WorldACL(zk.PermAll)
)

type request struct {
	etcdv2Op etcdv2Op
	etcdv3Op clientv3.Op
	zkOp     zkOp
	consulOp consulOp
}

type etcdv2Op struct {
	key   string
	value string
}

type zkOp struct {
	key   string
	value []byte
}

type consulOp struct {
	key   string
	value []byte
}

var (
	// dialTotal counts the number of mustCreateConn calls so that endpoint
	// connections can be handed out in round-robin order
	dialTotal int
)

func mustCreateConnEtcdv3(endpoints []string) *clientv3.Client {
	endpoint := endpoints[dialTotal%len(endpoints)]
	dialTotal++
	cfg := clientv3.Config{Endpoints: []string{endpoint}}
	client, err := clientv3.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error: %v\n", err)
		os.Exit(1)
	}
	return client
}

func mustCreateClientsEtcdv3(endpoints []string, totalClients, totalConns int) []*clientv3.Client {
	conns := make([]*clientv3.Client, totalConns)
	for i := range conns {
		conns[i] = mustCreateConnEtcdv3(endpoints)
	}

	clients := make([]*clientv3.Client, totalClients)
	for i := range clients {
		clients[i] = conns[i%int(totalConns)]
	}
	return clients
}

func mustCreateClientsEtcdv2(endpoints []string, total int) []clientv2.KeysAPI {
	cks := make([]clientv2.KeysAPI, total)
	for i := range cks {
		endpoint := endpoints[dialTotal%len(endpoints)]
		dialTotal++

		if !strings.HasPrefix(endpoint, "http://") {
			endpoint = "http://" + endpoint
		}
		cfg := clientv2.Config{
			Endpoints:               []string{endpoint},
			Transport:               clientv2.DefaultTransport,
			HeaderTimeoutPerRequest: time.Second,
		}
		c, err := clientv2.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		kapi := clientv2.NewKeysAPI(c)

		cks[i] = kapi
	}
	return cks
}

func mustCreateConnsZk(endpoints []string, total int) []*zk.Conn {
	zks := make([]*zk.Conn, total)
	for i := range zks {
		endpoint := endpoints[dialTotal%len(endpoints)]
		dialTotal++
		conn, _, err := zk.Connect([]string{endpoint}, time.Second)
		if err != nil {
			log.Fatal(err)
		}
		zks[i] = conn
	}
	return zks
}

func mustCreateConnsConsul(endpoints []string, total int) []*consulapi.KV {
	css := make([]*consulapi.KV, total)
	for i := range css {
		endpoint := endpoints[dialTotal%len(endpoints)]
		dialTotal++

		dcfg := consulapi.DefaultConfig()
		dcfg.Address = endpoint // x.x.x.x:8500
		cli, err := consulapi.NewClient(dcfg)
		if err != nil {
			log.Fatal(err)
		}

		css[i] = cli.KV()
	}
	return css
}

func mustRandBytes(n int) []byte {
	rb := make([]byte, n)
	_, err := rand.Read(rb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate value: %v\n", err)
		os.Exit(1)
	}
	return rb
}

func doPutEtcdv2(ctx context.Context, conn clientv2.KeysAPI, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.etcdv2Op
		st := time.Now()

		_, err := conn.Set(context.Background(), op.key, op.value, nil)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func getTotalKeysEtcdv2(endpoints []string) map[string]int64 {
	rs := make(map[string]int64)
	for _, ep := range endpoints {
		rs[ep] = 0 // not supported in metrics
	}
	return rs
}

func doPutEtcdv3(ctx context.Context, client clientv3.KV, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.etcdv3Op
		st := time.Now()
		_, err := client.Do(ctx, op)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func getTotalKeysEtcdv3(endpoints []string) map[string]int64 {
	rs := make(map[string]int64)
	for _, ep := range endpoints {
		if !strings.HasPrefix(ep, "http://") {
			ep = "http://" + ep
		}
		resp, err := http.Get(ep + "/metrics")
		if err != nil {
			log.Println(err)
			rs[ep] = 0
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			txt := scanner.Text()
			if strings.HasPrefix(txt, "#") {
				continue
			}
			ts := strings.SplitN(txt, " ", 2)
			fv := 0.0
			if len(ts) == 2 {
				v, err := strconv.ParseFloat(ts[1], 64)
				if err == nil {
					fv = v
				}
			}
			if ts[0] == "etcd_storage_keys_total" {
				rs[ep] = int64(fv)
				break
			}
		}
		gracefulClose(resp)
	}
	return rs
}

func doPutZk(conn *zk.Conn, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.zkOp
		st := time.Now()

		_, err := conn.Create(op.key, op.value, zkCreateFlags, zkCreateAcl)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func getTotalKeysZk(endpoints []string) map[string]int64 {
	rs := make(map[string]int64)
	stats, ok := zk.FLWSrvr(endpoints, 5*time.Second)
	if !ok {
		log.Printf("getTotalKeysZk failed with %+v", stats)
		for _, ep := range endpoints {
			rs[ep] = 0
		}
		return rs
	}
	for i, s := range stats {
		rs[endpoints[i]] = s.NodeCount
	}
	return rs
}

func doPutConsul(conn *consulapi.KV, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.consulOp
		st := time.Now()

		_, err := conn.Put(&consulapi.KVPair{Key: op.key, Value: op.value}, nil)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func getTotalKeysConsul(endpoints []string) map[string]int64 {
	rs := make(map[string]int64)
	for _, ep := range endpoints {
		rs[ep] = 0 // not supported in consul
	}
	return rs
}

func compactKV(clients []*clientv3.Client) {
	var curRev int64
	for _, c := range clients {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := c.KV.Get(ctx, "foo")
		cancel()
		if err != nil {
			panic(err)
		}
		curRev = resp.Header.Revision
		break
	}
	revToCompact := max(0, curRev-1000)
	for _, c := range clients {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.KV.Compact(ctx, revToCompact)
		cancel()
		if err != nil {
			panic(err)
		}
		break
	}
}

func doRangeEtcdv2(conn clientv2.KeysAPI, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.etcdv2Op

		st := time.Now()
		_, err := conn.Get(context.Background(), op.key, nil)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func doRangeEtcdv3(client clientv3.KV, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.etcdv3Op

		st := time.Now()
		_, err := client.Do(context.Background(), op)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func doRangeZk(conn *zk.Conn, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.zkOp

		st := time.Now()
		_, _, err := conn.Get(op.key)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func doRangeConsul(conn *consulapi.KV, requests <-chan request) {
	defer wg.Done()

	for req := range requests {
		op := req.consulOp

		st := time.Now()
		_, _, err := conn.Get(op.key, &consulapi.QueryOptions{AllowStale: true})

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		results <- result{errStr: errStr, duration: time.Since(st), happened: time.Now()}
		bar.Increment()
	}
}

func max(n1, n2 int64) int64 {
	if n1 > n2 {
		return n1
	}
	return n2
}

func randBytes(bytesN int) []byte {
	const (
		letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	src := mrand.NewSource(time.Now().UnixNano())
	b := make([]byte, bytesN)
	for i, cache, remain := bytesN-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}

func multiRandStrings(keyN, sliceN int) []string {
	m := make(map[string]struct{})
	for len(m) != sliceN {
		m[string(randBytes(keyN))] = struct{}{}
	}
	rs := make([]string, sliceN)
	idx := 0
	for k := range m {
		rs[idx] = k
		idx++
	}
	return rs
}

func toFile(txt, fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	_, err = f.WriteString(txt)
	return err
}

func toMillisecond(d time.Duration) float64 {
	return d.Seconds() * 1000
}

// gracefulClose drains http.Response.Body until it hits EOF
// and closes it. This prevents TCP/TLS connections from closing,
// therefore available for reuse.
func gracefulClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}
