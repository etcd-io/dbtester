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

// dbtester is distributed database tester.
//
//	Usage:
//	  dbtester [command]
//
//	Available Commands:
//	  agent       Database agent in remote servers.
//	  analyze     Analyzes test results specific to dbtester.
//	  bench       Low-level benchmark tool for etcd, Zookeeper, etcd2, consul.
//	  start       Starts database through RPC calls.
//	  stop        Stops database through RPC calls.
//	  restart     Restarts database through RPC calls.
//	  upload      Uploads to cloud storage.
//
//	Flags:
//	  -h, --help   help for dbtester
//
//	Use "dbtester [command] --help" for more information about a command.
//
package main

import (
	"fmt"
	"os"

	"github.com/coreos/dbtester/agent"
	"github.com/coreos/dbtester/analyze"
	"github.com/coreos/dbtester/bench"
	"github.com/coreos/dbtester/control"
	"github.com/coreos/dbtester/script"
	"github.com/coreos/dbtester/upload"

	"github.com/spf13/cobra"
)

var (
	rootCommand = &cobra.Command{
		Use:        "dbtester",
		Short:      "dbtester is distributed database tester.",
		SuggestFor: []string{"dbtstetr", "dbtes"},
	}
)

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCommand.AddCommand(agent.Command)
	rootCommand.AddCommand(analyze.Command)
	rootCommand.AddCommand(bench.Command)
	rootCommand.AddCommand(control.StartCommand)
	rootCommand.AddCommand(control.StopCommand)
	rootCommand.AddCommand(control.RestartCommand)
	rootCommand.AddCommand(script.Command)
	rootCommand.AddCommand(upload.Command)
}

func main() {
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}
