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

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/dbtester/dbtesterpb"
	"github.com/coreos/dbtester/pkg/fileinspect"

	"github.com/gyuho/linux-inspect/psn"
	"golang.org/x/net/context"
)

// implements dbtesterpb.TransporterServer
type transporterServer struct {
	req dbtesterpb.Request

	databaseLogFile      *os.File
	proxyDatabaseLogfile *os.File
	clientNumPath        string

	// cmd is the main process that's running the database
	cmd *exec.Cmd
	// cmdWait channel is closed
	// after database process is closed
	cmdWait chan struct{}

	pid int64

	proxyCmd     *exec.Cmd
	proxyCmdWait chan struct{}
	proxyPid     int64

	metricsCSV *psn.CSV

	// trigger log uploads to cloud storage
	// this should be triggered before we shut down
	// the agent server
	uploadSig chan struct{}
	csvReady  chan struct{}

	// notified after all tests finish
	notifier chan os.Signal
}

// NewServer returns a new server that implements gRPC interface.
func NewServer() dbtesterpb.TransporterServer {
	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, syscall.SIGINT, syscall.SIGTERM)

	return &transporterServer{
		clientNumPath: globalFlags.clientNumPath,
		uploadSig:     make(chan struct{}, 1),
		csvReady:      make(chan struct{}),
		notifier:      notifier,
	}
}

func (t *transporterServer) Transfer(ctx context.Context, req *dbtesterpb.Request) (*dbtesterpb.Response, error) {
	if req != nil {
		plog.Infof("received gRPC request %q with database %q (clients: %d)", req.Operation, req.DatabaseID, req.CurrentClientNumber)
	}

	if req.Operation == dbtesterpb.Operation_Start {
		f, err := openToAppend(globalFlags.databaseLog)
		if err != nil {
			return nil, err
		}
		t.databaseLogFile = f

		plog.Infof("agent log path: %q", globalFlags.agentLog)
		plog.Infof("database log path: %q", globalFlags.databaseLog)
		if req.DatabaseID == dbtesterpb.DatabaseID_zetcd__beta || req.DatabaseID == dbtesterpb.DatabaseID_cetcd__beta {
			proxyLog := globalFlags.databaseLog + "-" + t.req.DatabaseID.String()
			pf, err := openToAppend(proxyLog)
			if err != nil {
				return nil, err
			}
			t.proxyDatabaseLogfile = pf
			plog.Infof("proxy-database log path: %q", proxyLog)
		}
		plog.Infof("system metrics CSV path: %q", globalFlags.systemMetricsCSV)

		switch req.DatabaseID {
		case dbtesterpb.DatabaseID_zookeeper__r3_4_9:
			plog.Infof("Zookeeper working directory: %q", globalFlags.zkWorkDir)
			plog.Infof("Zookeeper data directory: %q", globalFlags.zkDataDir)
			plog.Infof("Zookeeper configuration path: %q", globalFlags.zkConfig)

		case dbtesterpb.DatabaseID_etcd__v2_3, dbtesterpb.DatabaseID_etcd__tip:
			plog.Infof("etcd executable binary path: %q", globalFlags.etcdExec)
			plog.Infof("etcd data directory: %q", globalFlags.etcdDataDir)

		case dbtesterpb.DatabaseID_zetcd__beta:
			plog.Infof("zetcd executable binary path: %q", globalFlags.zetcdExec)
			plog.Infof("zetcd data directory: %q", globalFlags.etcdDataDir)

		case dbtesterpb.DatabaseID_cetcd__beta:
			plog.Infof("cetcd executable binary path: %q", globalFlags.cetcdExec)
			plog.Infof("cetcd data directory: %q", globalFlags.etcdDataDir)

		case dbtesterpb.DatabaseID_consul__v0_7_5:
			plog.Infof("Consul executable binary path: %q", globalFlags.consulExec)
			plog.Infof("Consul data directory: %q", globalFlags.consulDataDir)
		}

		// re-use configurations for next requests
		t.req = *req
	}
	if req.Operation == dbtesterpb.Operation_Heartbeat {
		t.req.CurrentClientNumber = req.CurrentClientNumber
	}

	var diskSpaceUsageBytes int64
	switch req.Operation {
	case dbtesterpb.Operation_Start:
		switch t.req.DatabaseID {
		case dbtesterpb.DatabaseID_etcd__v2_3,
			dbtesterpb.DatabaseID_etcd__v3_1,
			dbtesterpb.DatabaseID_etcd__v3_2,
			dbtesterpb.DatabaseID_etcd__tip,
			dbtesterpb.DatabaseID_zetcd__beta,
			dbtesterpb.DatabaseID_cetcd__beta:
			if err := startEtcd(&globalFlags, t); err != nil {
				plog.Errorf("startEtcd error %v", err)
				return nil, err
			}
			switch t.req.DatabaseID {
			case dbtesterpb.DatabaseID_zetcd__beta:
				if err := startZetcd(&globalFlags, t); err != nil {
					plog.Errorf("startZetcd error %v", err)
					return nil, err
				}
				go func() {
					defer close(t.proxyCmdWait)
					if err := t.proxyCmd.Wait(); err != nil {
						plog.Errorf("cmd.Wait %q returned error %v", t.proxyCmd.Path, err)
						return
					}
					plog.Infof("exiting %q", t.proxyCmd.Path)
				}()
			case dbtesterpb.DatabaseID_cetcd__beta:
				if err := startCetcd(&globalFlags, t); err != nil {
					plog.Errorf("startCetcd error %v", err)
					return nil, err
				}
				go func() {
					defer close(t.proxyCmdWait)
					if err := t.proxyCmd.Wait(); err != nil {
						plog.Errorf("cmd.Wait %q returned error %v", t.proxyCmd.Path, err)
						return
					}
					plog.Infof("exiting %q", t.proxyCmd.Path)
				}()
			}
		case dbtesterpb.DatabaseID_zookeeper__r3_4_9,
			dbtesterpb.DatabaseID_zookeeper__r3_5_2_alpha:
			if err := startZookeeper(&globalFlags, t); err != nil {
				plog.Errorf("startZookeeper error %v", err)
				return nil, err
			}
		case dbtesterpb.DatabaseID_consul__v0_7_5,
			dbtesterpb.DatabaseID_consul__v0_8_0:
			if err := startConsul(&globalFlags, t); err != nil {
				plog.Errorf("startConsul error %v", err)
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown database %q", t.req.DatabaseID)
		}

		go func() {
			defer close(t.cmdWait)
			if err := t.cmd.Wait(); err != nil {
				plog.Errorf("cmd.Wait %q returned error %v", t.cmd.Path, err)
				return
			}
			plog.Infof("exiting %q", t.cmd.Path)
		}()

		if err := startMetrics(&globalFlags, t); err != nil {
			plog.Errorf("startMetrics error %v", err)
			return nil, err
		}

	case dbtesterpb.Operation_Stop:
		if t.cmd == nil {
			return nil, fmt.Errorf("nil command")
		}

		// to collect more monitoring data
		plog.Infof("waiting a few more seconds before stopping %q", t.cmd.Path)
		time.Sleep(3 * time.Second)

		plog.Infof("sending %q to %q [PID: %d]", syscall.SIGINT, t.cmd.Path, t.pid)
		if err := t.cmd.Process.Signal(syscall.SIGINT); err != nil {
			return nil, err
		}
		time.Sleep(3 * time.Second)
		plog.Infof("sending %q to %q [PID: %d]", syscall.SIGTERM, t.cmd.Path, t.pid)
		if err := syscall.Kill(int(t.pid), syscall.SIGTERM); err != nil {
			plog.Warningf("syscall.Kill failed with %v", err)
		}
		<-t.cmdWait
		if t.databaseLogFile != nil {
			t.databaseLogFile.Sync()
			t.databaseLogFile.Close()
		}
		plog.Infof("stopped binary %q [PID: %d]", t.req.DatabaseID.String(), t.pid)

		if t.proxyCmd != nil {
			plog.Infof("sending %q to %q [PID: %d]", syscall.SIGINT, t.proxyCmd.Path, t.proxyPid)
			if err := t.proxyCmd.Process.Signal(syscall.SIGINT); err != nil {
				return nil, err
			}
			time.Sleep(3 * time.Second)
			plog.Infof("sending %q to %q [PID: %d]", syscall.SIGTERM, t.proxyCmd.Path, t.proxyPid)
			if err := syscall.Kill(int(t.proxyPid), syscall.SIGTERM); err != nil {
				plog.Warningf("syscall.Kill failed with %v", err)
			}
			<-t.proxyCmdWait
			plog.Infof("stopped binary proxy for %q [PID: %d]", t.req.DatabaseID.String(), t.pid)
		}
		if t.proxyDatabaseLogfile != nil {
			t.proxyDatabaseLogfile.Sync()
			t.proxyDatabaseLogfile.Close()
		}

		t.uploadSig <- struct{}{}
		<-t.csvReady

		if t.req.TriggerLogUpload {
			if err := uploadLog(&globalFlags, t); err != nil {
				plog.Warningf("uploadLog error %v", err)
				return nil, err
			}
		}

		dbs, err := measureDatabasSize(globalFlags, req.DatabaseID)
		if err != nil {
			plog.Warningf("measureDatabasSize error %v", err)
			return nil, err
		}
		diskSpaceUsageBytes = dbs

	case dbtesterpb.Operation_Heartbeat:
		plog.Infof("overwriting clients num %d to %q", t.req.CurrentClientNumber, t.clientNumPath)
		if err := toFile(fmt.Sprintf("%d", t.req.CurrentClientNumber), t.clientNumPath); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("Not implemented %v", req.Operation)
	}

	plog.Info("Transfer success!")
	return &dbtesterpb.Response{Success: true, DiskSpaceUsageBytes: diskSpaceUsageBytes}, nil
}

func measureDatabasSize(flg flags, rdb dbtesterpb.DatabaseID) (int64, error) {
	switch rdb {
	case dbtesterpb.DatabaseID_etcd__v2_3:
		return fileinspect.Size(flg.etcdDataDir)
	case dbtesterpb.DatabaseID_etcd__tip:
		return fileinspect.Size(flg.etcdDataDir)
	case dbtesterpb.DatabaseID_zookeeper__r3_4_9:
		return fileinspect.Size(flg.zkDataDir)
	case dbtesterpb.DatabaseID_consul__v0_7_5:
		return fileinspect.Size(flg.consulDataDir)
	case dbtesterpb.DatabaseID_cetcd__beta:
		return fileinspect.Size(flg.etcdDataDir)
	case dbtesterpb.DatabaseID_zetcd__beta:
		return fileinspect.Size(flg.etcdDataDir)
	default:
		return 0, fmt.Errorf("uknown %q", rdb)
	}
}
