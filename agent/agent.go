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

package agent

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/dbtester/remotestorage"
	"github.com/gyuho/psn/ps"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type (
	Flags struct {
		GRPCPort         string
		WorkingDirectory string
	}

	// ZookeeperConfig is zookeeper configuration.
	// http://zookeeper.apache.org/doc/trunk/zookeeperAdmin.html
	ZookeeperConfig struct {
		TickTime       int
		DataDir        string
		ClientPort     string
		InitLimit      int
		SyncLimit      int
		PreAllocSize   int64
		MaxClientCnxns int64
		Peers          []ZookeeperPeer
	}
	ZookeeperPeer struct {
		MyID int
		IP   string
	}

	ConsulConfig struct {
		Bootstrap     bool     `json:"bootstrap"`
		Server        bool     `json:"server,omitempty"`
		AdvertiseAddr string   `json:"advertise_addr,omitempty"`
		DataCenter    string   `json:"datacenter,omitempty"`
		DataDir       string   `json:"data_dir,omitempty"`
		Encrypt       string   `json:"encrypt,omitempty"`
		LogLevel      string   `json:"log_level,omitempty"`
		EnableSyslog  bool     `json:"enable_syslog,omitempty"`
		StartJoin     []string `json:"start_join,omitempty"`
		RetryJoin     []string `json:"retry_join,omitempty"`
		RetryInterval string   `json:"retry_interval,omitempty"`
	}
)

var (
	shell = os.Getenv("SHELL")

	agentLogPath = "agent.log"

	etcdBinaryPath   = filepath.Join(os.Getenv("GOPATH"), "bin/etcd")
	consulBinaryPath = filepath.Join(os.Getenv("GOPATH"), "bin/consul")
	javaBinaryPath   = "/usr/bin/java"

	etcdToken     = "etcd_token"
	etcdDataDir   = "data.etcd"
	consulDataDir = "data.consul"

	zkWorkingDir = "zookeeper"
	zkDataDir    = "zookeeper/data.zk"
	zkConfigPath = "zookeeper.config"
	zkTemplate   = `tickTime={{.TickTime}}
dataDir={{.DataDir}}
clientPort={{.ClientPort}}
initLimit={{.InitLimit}}
syncLimit={{.SyncLimit}}
preAllocSize={{.PreAllocSize}}
maxClientCnxns={{.MaxClientCnxns}}
{{range .Peers}}server.{{.MyID}}={{.IP}}:2888:3888
{{end}}
`
	zkConfigDefault = ZookeeperConfig{
		TickTime:       2000,
		ClientPort:     "2181",
		InitLimit:      5,
		SyncLimit:      5,
		PreAllocSize:   65536 * 1024,
		MaxClientCnxns: 60,
		Peers: []ZookeeperPeer{
			{MyID: 1, IP: ""},
			{MyID: 2, IP: ""},
			{MyID: 3, IP: ""},
		},
	}

	Command = &cobra.Command{
		Use:   "agent",
		Short: "Database agent in remote servers.",
		RunE:  CommandFunc,
	}

	globalFlags = Flags{}
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	if len(shell) == 0 {
		shell = "sh"
	}
	Command.PersistentFlags().StringVar(&globalFlags.GRPCPort, "agent-port", ":3500", "Port to server agent gRPC server.")
	Command.PersistentFlags().StringVar(&globalFlags.WorkingDirectory, "working-directory", homeDir(), "Working directory.")
}

func CommandFunc(cmd *cobra.Command, args []string) error {
	if !exist(globalFlags.WorkingDirectory) {
		return fmt.Errorf("%s does not exist", globalFlags.WorkingDirectory)
	}
	if !filepath.HasPrefix(agentLogPath, globalFlags.WorkingDirectory) {
		agentLogPath = filepath.Join(globalFlags.WorkingDirectory, agentLogPath)
	}

	f, err := openToAppend(agentLogPath)
	if err != nil {
		return err
	}
	defer f.Close()
	log.SetOutput(f)

	log.Printf("gRPC serving %s", globalFlags.GRPCPort)
	var (
		grpcServer = grpc.NewServer()
		sender     = NewTransporterServer()
	)
	ln, err := net.Listen("tcp", globalFlags.GRPCPort)
	if err != nil {
		return err
	}

	RegisterTransporterServer(grpcServer, sender)

	if err := grpcServer.Serve(ln); err != nil {
		return err
	}
	return nil
}

type transporterServer struct { // satisfy TransporterServer
	req     Request
	cmd     *exec.Cmd
	logfile *os.File
	pid     int
}

var databaseStopped = make(chan struct{})

func (t *transporterServer) Transfer(ctx context.Context, r *Request) (*Response, error) {
	peerIPs := strings.Split(r.PeerIPs, "___")
	if r.Operation == Request_Start || r.Operation == Request_Restart {
		if !filepath.HasPrefix(etcdDataDir, globalFlags.WorkingDirectory) {
			etcdDataDir = filepath.Join(globalFlags.WorkingDirectory, etcdDataDir)
		}
		if !filepath.HasPrefix(consulDataDir, globalFlags.WorkingDirectory) {
			consulDataDir = filepath.Join(globalFlags.WorkingDirectory, consulDataDir)
		}
		if !filepath.HasPrefix(zkWorkingDir, globalFlags.WorkingDirectory) {
			zkWorkingDir = filepath.Join(globalFlags.WorkingDirectory, zkWorkingDir)
		}
		if !filepath.HasPrefix(zkDataDir, globalFlags.WorkingDirectory) {
			zkDataDir = filepath.Join(globalFlags.WorkingDirectory, zkDataDir)
		}
		if !filepath.HasPrefix(r.DatabaseLogPath, globalFlags.WorkingDirectory) {
			r.DatabaseLogPath = filepath.Join(globalFlags.WorkingDirectory, r.DatabaseLogPath)
		}
		if !filepath.HasPrefix(r.MonitorResultPath, globalFlags.WorkingDirectory) {
			r.MonitorResultPath = filepath.Join(globalFlags.WorkingDirectory, r.MonitorResultPath)
		}

		log.Printf("Working directory: %s", globalFlags.WorkingDirectory)
		log.Printf("etcd data directory: %s", etcdDataDir)
		log.Printf("Consul data directory: %s", consulDataDir)
		log.Printf("Zookeeper working directory: %s", zkWorkingDir)
		log.Printf("Zookeeper data directory: %s", zkDataDir)
		log.Printf("Database log path: %s", r.DatabaseLogPath)
		log.Printf("Monitor result path: %s", r.MonitorResultPath)
	}
	if r.Operation == Request_Start {
		t.req = *r
	}

	if t.req.StorageKey != "" {
		if err := toFile(t.req.StorageKey, filepath.Join(globalFlags.WorkingDirectory, "key.json")); err != nil {
			return nil, err
		}
	}

	var processPID int
	switch r.Operation {
	case Request_Start:
		switch t.req.Database {
		case Request_etcdv2, Request_etcdv3:
			_, err := os.Stat(etcdBinaryPath)
			if err != nil {
				return nil, err
			}
			if err := os.RemoveAll(etcdDataDir); err != nil {
				return nil, err
			}
			f, err := openToAppend(t.req.DatabaseLogPath)
			if err != nil {
				return nil, err
			}
			t.logfile = f

			clusterN := len(peerIPs)
			names := make([]string, clusterN)
			clientURLs := make([]string, clusterN)
			peerURLs := make([]string, clusterN)
			members := make([]string, clusterN)
			for i, u := range peerIPs {
				names[i] = fmt.Sprintf("etcd-%d", i+1)
				clientURLs[i] = fmt.Sprintf("http://%s:2379", u)
				peerURLs[i] = fmt.Sprintf("http://%s:2380", u)
				members[i] = fmt.Sprintf("%s=%s", names[i], peerURLs[i])
			}
			clusterStr := strings.Join(members, ",")
			flags := []string{
				"--name", names[t.req.ServerIndex],
				"--data-dir", etcdDataDir,

				"--listen-client-urls", clientURLs[t.req.ServerIndex],
				"--advertise-client-urls", clientURLs[t.req.ServerIndex],

				"--listen-peer-urls", peerURLs[t.req.ServerIndex],
				"--initial-advertise-peer-urls", peerURLs[t.req.ServerIndex],

				"--initial-cluster-token", etcdToken,
				"--initial-cluster", clusterStr,
				"--initial-cluster-state", "new",
			}
			flagString := strings.Join(flags, " ")

			cmd := exec.Command(etcdBinaryPath, flags...)
			cmd.Stdout = f
			cmd.Stderr = f
			log.Printf("Starting: %s %s", cmd.Path, flagString)
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			t.cmd = cmd
			t.pid = cmd.Process.Pid
			log.Printf("Started: %s [PID: %d]", cmd.Path, t.pid)
			processPID = t.pid
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Printf("%s cmd.Wait returned %v", cmd.Path, err)
					return
				}
				log.Printf("Exiting %s", cmd.Path)
			}()

		case Request_ZooKeeper:
			_, err := os.Stat(javaBinaryPath)
			if err != nil {
				return nil, err
			}

			log.Printf("os.Chdir: %s", zkWorkingDir)
			if err := os.Chdir(zkWorkingDir); err != nil {
				return nil, err
			}

			log.Printf("os.MkdirAll: %s", zkDataDir)
			if err := os.MkdirAll(zkDataDir, 0777); err != nil {
				return nil, err
			}

			idFilePath := filepath.Join(zkDataDir, "myid")
			log.Printf("Writing %d to %s", t.req.ZookeeperMyID, idFilePath)
			if err := toFile(fmt.Sprintf("%d", t.req.ZookeeperMyID), idFilePath); err != nil {
				return nil, err
			}

			// generate zookeeper config
			zkCfg := zkConfigDefault
			zkCfg.DataDir = zkDataDir
			peers := []ZookeeperPeer{}
			for i := range peerIPs {
				peers = append(peers, ZookeeperPeer{MyID: i + 1, IP: peerIPs[i]})
			}
			zkCfg.Peers = peers
			zkCfg.PreAllocSize = t.req.ZookeeperPreAllocSize
			zkCfg.MaxClientCnxns = t.req.ZookeeperMaxClientCnxns
			tpl := template.Must(template.New("zkTemplate").Parse(zkTemplate))
			buf := new(bytes.Buffer)
			if err := tpl.Execute(buf, zkCfg); err != nil {
				return nil, err
			}
			zc := buf.String()

			configFilePath := filepath.Join(zkWorkingDir, zkConfigPath)
			log.Printf("Writing %q to %s", zc, configFilePath)
			if err := toFile(zc, configFilePath); err != nil {
				return nil, err
			}

			f, err := openToAppend(t.req.DatabaseLogPath)
			if err != nil {
				return nil, err
			}
			t.logfile = f

			// this changes for different releases
			flagString := `-cp zookeeper-3.4.8.jar:lib/slf4j-api-1.6.1.jar:lib/slf4j-log4j12-1.6.1.jar:lib/log4j-1.2.16.jar:conf org.apache.zookeeper.server.quorum.QuorumPeerMain`
			args := []string{shell, "-c", javaBinaryPath + " " + flagString + " " + configFilePath}

			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = f
			cmd.Stderr = f
			log.Printf("Starting: %s %s", cmd.Path, strings.Join(args[1:], " "))
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			t.cmd = cmd
			t.pid = cmd.Process.Pid
			log.Printf("Started: %s [PID: %d]", cmd.Path, t.pid)
			processPID = t.pid
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Printf("%s cmd.Wait returned %v", cmd.Path, err)
					return
				}
				log.Printf("Exiting %s", cmd.Path)
			}()

		case Request_Consul:
			_, err := os.Stat(consulBinaryPath)
			if err != nil {
				return nil, err
			}
			if err := os.RemoveAll(consulDataDir); err != nil {
				return nil, err
			}
			f, err := openToAppend(t.req.DatabaseLogPath)
			if err != nil {
				return nil, err
			}
			t.logfile = f

			var flags []string
			if t.req.ServerIndex == 0 { // leader
				flags = []string{
					"agent",
					"-server",
					"-data-dir", consulDataDir,
					"-bind", peerIPs[t.req.ServerIndex],
					"-client", peerIPs[t.req.ServerIndex],
					"-bootstrap-expect", "3",
				}
			} else {
				flags = []string{
					"agent",
					"-server",
					"-data-dir", consulDataDir,
					"-bind", peerIPs[t.req.ServerIndex],
					"-client", peerIPs[t.req.ServerIndex],
					"-join", peerIPs[0],
				}
			}
			flagString := strings.Join(flags, " ")

			cmd := exec.Command(consulBinaryPath, flags...)
			cmd.Stdout = f
			cmd.Stderr = f
			log.Printf("Starting: %s %s", cmd.Path, flagString)
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			t.cmd = cmd
			t.pid = cmd.Process.Pid
			log.Printf("Started: %s [PID: %d]", cmd.Path, t.pid)
			processPID = t.pid
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Printf("%s cmd.Wait returned %v", cmd.Path, err)
					return
				}
				log.Printf("Exiting %s", cmd.Path)
			}()

		default:
			return nil, fmt.Errorf("unknown database (%q)", r.Database)
		}

	case Request_Restart:
		if t.cmd == nil {
			return nil, fmt.Errorf("nil command")
		}

		log.Printf("Restarting %s", t.req.Database)
		if r.Database == Request_ZooKeeper {
			if err := os.Chdir(zkWorkingDir); err != nil {
				return nil, err
			}
		}

		f, err := openToAppend(t.req.DatabaseLogPath)
		if err != nil {
			return nil, err
		}
		t.logfile = f

		cmd := exec.Command(t.cmd.Path, t.cmd.Args[1:]...)
		cmd.Stdout = f
		cmd.Stderr = f
		log.Printf("Restarting: %s", strings.Join(t.cmd.Args, " "))
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		t.cmd = cmd
		t.pid = cmd.Process.Pid
		log.Printf("Restarted: %s [PID: %d]", cmd.Path, t.pid)
		processPID = t.pid
		go func() {
			if err := cmd.Wait(); err != nil {
				log.Printf("%s cmd.Wait returned %v", cmd.Path, err)
				return
			}
			log.Printf("Exiting %s", cmd.Path)
		}()

	case Request_Stop:
		if t.cmd == nil {
			return nil, fmt.Errorf("nil command")
		}
		log.Printf("Stopping %s [PID: %d]", t.req.Database, t.pid)
		if err := syscall.Kill(t.pid, syscall.SIGTERM); err != nil {
			return nil, err
		}
		if t.logfile != nil {
			t.logfile.Close()
		}
		log.Printf("Stopped: %s [PID: %d]", t.cmd.Path, t.pid)
		processPID = t.pid
		databaseStopped <- struct{}{}

	default:
		return nil, fmt.Errorf("Not implemented %v", r.Operation)
	}

	if r.Operation == Request_Start || r.Operation == Request_Restart {
		go func(processPID int) {
			notifier := make(chan os.Signal, 1)
			signal.Notify(notifier, syscall.SIGINT, syscall.SIGTERM)

			rFunc := func() error {
				pss, err := ps.List(&ps.Process{Stat: ps.Stat{Pid: int64(processPID)}})
				if err != nil {
					return err
				}

				f, err := openToAppend(t.req.MonitorResultPath)
				if err != nil {
					return err
				}
				defer f.Close()

				return ps.WriteToCSV(f, pss...)
			}

			log.Printf("%s monitor saved at %s", t.req.Database, t.req.MonitorResultPath)
			var err error
			if err = rFunc(); err != nil {
				log.Warningln("error:", err)
				return
			}

		escape:
			for {
				select {
				case <-time.After(time.Second):
					if err = rFunc(); err != nil {
						log.Warnf("Monitoring error %v", err)
						break escape
					}
				case sig := <-notifier:
					log.Printf("Received %v", sig)
					return
				case <-databaseStopped:
					log.Println("Monitoring stopped. Uploading data to cloud storage...")
					u, err := remotestorage.NewGoogleCloudStorage([]byte(t.req.StorageKey), t.req.GoogleCloudProjectName)
					if err != nil {
						log.Warnf("error (%v)", err)
						return
					}

					// set up file names
					srcDatabaseLogPath := t.req.DatabaseLogPath
					dstDatabaseLogPath := filepath.Base(t.req.DatabaseLogPath)
					if !strings.HasPrefix(filepath.Base(t.req.DatabaseLogPath), t.req.LogPrefix) {
						dstDatabaseLogPath = fmt.Sprintf("%s-%d-%s", t.req.LogPrefix, t.req.ServerIndex+1, filepath.Base(t.req.DatabaseLogPath))
					}
					log.Printf("Uploading %s to %s", srcDatabaseLogPath, dstDatabaseLogPath)
					var uerr error
					for k := 0; k < 5; k++ {
						if uerr = u.UploadFile(t.req.Bucket, srcDatabaseLogPath, dstDatabaseLogPath); uerr != nil {
							log.Println(uerr)
							continue
						} else {
							break
						}
					}

					srcMonitorResultPath := t.req.MonitorResultPath
					dstMonitorResultPath := filepath.Base(t.req.MonitorResultPath)
					if !strings.HasPrefix(filepath.Base(t.req.MonitorResultPath), t.req.LogPrefix) {
						dstMonitorResultPath = fmt.Sprintf("%s-%d-%s", t.req.LogPrefix, t.req.ServerIndex+1, filepath.Base(t.req.MonitorResultPath))
					}
					log.Printf("Uploading %s to %s", srcMonitorResultPath, dstMonitorResultPath)
					for k := 0; k < 5; k++ {
						if uerr = u.UploadFile(t.req.Bucket, srcMonitorResultPath, dstMonitorResultPath); uerr != nil {
							log.Println(uerr)
							continue
						} else {
							break
						}
					}

					srcAgentLogPath := agentLogPath
					dstAgentLogPath := filepath.Base(agentLogPath)
					if !strings.HasPrefix(filepath.Base(agentLogPath), t.req.LogPrefix) {
						dstAgentLogPath = fmt.Sprintf("%s-%d-%s", t.req.LogPrefix, t.req.ServerIndex+1, filepath.Base(agentLogPath))
					}
					log.Printf("Uploading %s to %s", srcAgentLogPath, dstAgentLogPath)
					for k := 0; k < 5; k++ {
						if uerr = u.UploadFile(t.req.Bucket, srcAgentLogPath, dstAgentLogPath); uerr != nil {
							log.Println(uerr)
							continue
						} else {
							break
						}
					}

					return
				}
			}
		}(processPID)
	}

	log.Printf("Transfer success!")
	return &Response{Success: true}, nil
}

func NewTransporterServer() TransporterServer {
	return &transporterServer{}
}
