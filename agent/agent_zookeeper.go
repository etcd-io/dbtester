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
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/dbtester/dbtesterpb"
)

var (
	zkTemplate = `tickTime={{.TickTime}}
dataDir={{.DataDir}}
clientPort={{.ClientPort}}
initLimit={{.InitLimit}}
syncLimit={{.SyncLimit}}
maxClientCnxns={{.MaxClientConnections}}
snapCount={{.SnapCount}}
{{range .Peers}}server.{{.MyID}}={{.IP}}:2888:3888
{{end}}
`
)

// ZookeeperConfig is zookeeper configuration.
// http://zookeeper.apache.org/doc/trunk/zookeeperAdmin.html
type ZookeeperConfig struct {
	TickTime             int64
	DataDir              string
	ClientPort           int64
	InitLimit            int64
	SyncLimit            int64
	MaxClientConnections int64
	SnapCount            int64
	Peers                []ZookeeperPeer
}

// ZookeeperPeer defines Zookeeper peer configuration.
type ZookeeperPeer struct {
	MyID int
	IP   string
}

var shell = os.Getenv("SHELL")

func init() {
	if len(shell) == 0 {
		shell = "sh"
	}
}

// Java class paths for Zookeeper.
// '-cp' is for 'class search path of directories and zip/jar files'.
// See https://zookeeper.apache.org/doc/trunk/zookeeperAdmin.html for more.
const (
	// JavaClassPathZookeeperr349 is the Java class paths of Zookeeper r3.4.9.
	// CHANGE THIS FOR DIFFERENT ZOOKEEPER RELEASE!
	// THIS IS ONLY VALID FOR Zookeeper r3.4.9.
	// Search correct paths with 'find ./zookeeper/lib | sort'.
	JavaClassPathZookeeperr349 = `-cp zookeeper-3.4.9.jar:lib/slf4j-api-1.6.1.jar:lib/slf4j-log4j12-1.6.1.jar:lib/log4j-1.2.16.jar:conf org.apache.zookeeper.server.quorum.QuorumPeerMain`

	// JavaClassPathZookeeperr352alpha is the Java class paths of Zookeeper r3.5.2-alpha.
	// CHANGE THIS FOR DIFFERENT ZOOKEEPER RELEASE!
	// THIS IS ONLY VALID FOR Zookeeper r3.5.2-alpha.
	// Search correct paths with 'find ./zookeeper/lib | sort'.
	JavaClassPathZookeeperr352alpha = `-cp zookeeper-3.5.2-alpha.jar:lib/slf4j-api-1.7.5.jar:lib/slf4j-log4j12-1.7.5.jar:lib/log4j-1.2.17.jar:conf org.apache.zookeeper.server.quorum.QuorumPeerMain`
)

// startZookeeper starts Zookeeper.
func startZookeeper(fs *flags, t *transporterServer) error {
	if !exist(fs.javaExec) {
		return fmt.Errorf("Java binary %q does not exist", globalFlags.javaExec)
	}

	if err := os.RemoveAll(fs.zkDataDir); err != nil {
		return err
	}
	if err := os.MkdirAll(fs.zkDataDir, 0777); err != nil {
		return err
	}

	// Zookeeper requires correct relative-path for runtime
	// needs manual 'cd' into the Zookeeper working directory!
	if err := os.Chdir(fs.zkWorkDir); err != nil {
		return err
	}

	ipath := filepath.Join(fs.zkDataDir, "myid")
	switch t.req.DatabaseID {
	case dbtesterpb.DatabaseID_zookeeper__r3_4_9:
		if t.req.Flag_Zookeeper_R3_4_9 == nil {
			return fmt.Errorf("request 'Flag_Zookeeper_R3_4_9' is nil")
		}
		plog.Infof("writing Zookeeper myid file %d to %s", t.req.Flag_Zookeeper_R3_4_9.MyID, ipath)
		if err := toFile(fmt.Sprintf("%d", t.req.Flag_Zookeeper_R3_4_9.MyID), ipath); err != nil {
			return err
		}
	case dbtesterpb.DatabaseID_zookeeper__r3_5_2_alpha:
		if t.req.Flag_Zookeeper_R3_5_2Alpha == nil {
			return fmt.Errorf("request 'Flag_Zookeeper_R3_5_2Alpha' is nil")
		}
		plog.Infof("writing Zookeeper myid file %d to %s", t.req.Flag_Zookeeper_R3_5_2Alpha.MyID, ipath)
		if err := toFile(fmt.Sprintf("%d", t.req.Flag_Zookeeper_R3_5_2Alpha.MyID), ipath); err != nil {
			return err
		}
	default:
		return fmt.Errorf("database ID %q is not supported", t.req.DatabaseID)
	}

	var cfg ZookeeperConfig
	peerIPs := strings.Split(t.req.PeerIPsString, "___")
	peers := []ZookeeperPeer{}
	for i := range peerIPs {
		peers = append(peers, ZookeeperPeer{MyID: i + 1, IP: peerIPs[i]})
	}
	switch t.req.DatabaseID {
	case dbtesterpb.DatabaseID_zookeeper__r3_4_9:
		cfg = ZookeeperConfig{
			TickTime:             t.req.Flag_Zookeeper_R3_4_9.TickTime,
			DataDir:              fs.zkDataDir,
			ClientPort:           t.req.Flag_Zookeeper_R3_4_9.ClientPort,
			InitLimit:            t.req.Flag_Zookeeper_R3_4_9.InitLimit,
			SyncLimit:            t.req.Flag_Zookeeper_R3_4_9.SyncLimit,
			MaxClientConnections: t.req.Flag_Zookeeper_R3_4_9.MaxClientConnections,
			Peers:                peers,
			SnapCount:            t.req.Flag_Zookeeper_R3_4_9.SnapCount,
		}
	case dbtesterpb.DatabaseID_zookeeper__r3_5_2_alpha:
		cfg = ZookeeperConfig{
			TickTime:             t.req.Flag_Zookeeper_R3_5_2Alpha.TickTime,
			DataDir:              fs.zkDataDir,
			ClientPort:           t.req.Flag_Zookeeper_R3_5_2Alpha.ClientPort,
			InitLimit:            t.req.Flag_Zookeeper_R3_5_2Alpha.InitLimit,
			SyncLimit:            t.req.Flag_Zookeeper_R3_5_2Alpha.SyncLimit,
			MaxClientConnections: t.req.Flag_Zookeeper_R3_5_2Alpha.MaxClientConnections,
			Peers:                peers,
			SnapCount:            t.req.Flag_Zookeeper_R3_5_2Alpha.SnapCount,
		}
	default:
		return fmt.Errorf("database ID %q is not supported", t.req.DatabaseID)
	}
	tpl := template.Must(template.New("zkTemplate").Parse(zkTemplate))
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, cfg); err != nil {
		return err
	}
	zctxt := buf.String()
	plog.Infof("writing Zookeeper config file %q (config %q)", fs.zkConfig, zctxt)
	if err := toFile(zctxt, fs.zkConfig); err != nil {
		return err
	}

	var flagString string
	switch t.req.DatabaseID {
	case dbtesterpb.DatabaseID_zookeeper__r3_4_9:
		if t.req.Flag_Zookeeper_R3_4_9.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Djute.maxbuffer=%d", t.req.Flag_Zookeeper_R3_4_9.JavaDJuteMaxBuffer)
		}
		if t.req.Flag_Zookeeper_R3_4_9.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Xms%s", t.req.Flag_Zookeeper_R3_4_9.JavaXms)
		}
		if t.req.Flag_Zookeeper_R3_4_9.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Xmx%s", t.req.Flag_Zookeeper_R3_4_9.JavaXmx)
		}
		// -Djute.maxbuffer=33554432 -Xms50G -Xmx50G
		if len(flagString) > 0 {
			flagString += " "
		}
		flagString += JavaClassPathZookeeperr349

	case dbtesterpb.DatabaseID_zookeeper__r3_5_2_alpha:
		if t.req.Flag_Zookeeper_R3_5_2Alpha.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Djute.maxbuffer=%d", t.req.Flag_Zookeeper_R3_5_2Alpha.JavaDJuteMaxBuffer)
		}
		if t.req.Flag_Zookeeper_R3_5_2Alpha.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Xms%s", t.req.Flag_Zookeeper_R3_5_2Alpha.JavaXms)
		}
		if t.req.Flag_Zookeeper_R3_5_2Alpha.JavaDJuteMaxBuffer != 0 {
			if len(flagString) > 0 {
				flagString += " "
			}
			flagString += fmt.Sprintf("-Xmx%s", t.req.Flag_Zookeeper_R3_5_2Alpha.JavaXmx)
		}
		// -Djute.maxbuffer=33554432 -Xms50G -Xmx50G
		if len(flagString) > 0 {
			flagString += " "
		}
		flagString += JavaClassPathZookeeperr352alpha

	default:
		return fmt.Errorf("database ID %q is not supported", t.req.DatabaseID)
	}

	args := []string{shell, "-c", fs.javaExec + " " + flagString + " " + fs.zkConfig}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = t.databaseLogFile
	cmd.Stderr = t.databaseLogFile
	cs := fmt.Sprintf("%s %s", cmd.Path, strings.Join(args[1:], " "))

	plog.Infof("starting database %q", cs)
	if err := cmd.Start(); err != nil {
		return err
	}
	t.cmd = cmd
	t.cmdWait = make(chan struct{})
	t.pid = int64(cmd.Process.Pid)

	plog.Infof("started database %q (PID: %d)", cs, t.pid)
	return nil
}
