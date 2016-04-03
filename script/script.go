package script

import (
	"bytes"
	"os"
	"text/template"

	"github.com/spf13/cobra"
)

type scriptConfig struct {
	DBName      string
	BucketName  string
	LogPrefix   string
	ClientPort  string
	ProjectName string
	KeyPath     string
	Conns       int
	Clients     int
	ValSize     int
	Total       int
}

var (
	Command = &cobra.Command{
		Use:        "script",
		Short:      "Generates cloud provisioning script.",
		SuggestFor: []string{"scrt"},
		RunE:       scriptCommandFunc,
	}

	outputPath string
	cfg        = scriptConfig{}
)

func init() {
	Command.PersistentFlags().StringVarP(&outputPath, "output", "o", "script.sh", "File path to store script.")
	Command.PersistentFlags().StringVarP(&cfg.DBName, "db-name", "d", "etcdv3", "Name of database (etcdv2, etcdv3, zookeeper, zk, consul).")
	Command.PersistentFlags().StringVarP(&cfg.BucketName, "bucket-name", "b", "", "Name of bucket to store results.")
	Command.PersistentFlags().StringVarP(&cfg.LogPrefix, "log-prefix", "p", "bench-01", "Prefix to name instances, logs files.")
	Command.PersistentFlags().StringVarP(&cfg.ClientPort, "client-port", "c", "2379", "2379 for etcd, 2181 for Zookeeper, 8500 for Consul.")
	Command.PersistentFlags().StringVarP(&cfg.ProjectName, "project-name", "n", "etcd-development", "Project name.")
	Command.PersistentFlags().StringVarP(&cfg.KeyPath, "key-path", "k", "$HOME/key.json", "Key path.")
	Command.PersistentFlags().IntVar(&cfg.Conns, "conns", 1, "conns.")
	Command.PersistentFlags().IntVar(&cfg.Clients, "clients", 1, "clients.")
	Command.PersistentFlags().IntVar(&cfg.ValSize, "val-size", 256, "val-size.")
	Command.PersistentFlags().IntVar(&cfg.Total, "total", 3000000, "total.")
}

func scriptCommandFunc(cmd *cobra.Command, args []string) error {
	tpl := template.Must(template.New("scriptTemplate").Parse(scriptTemplate))
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, cfg); err != nil {
		return err
	}
	return toFile(buf.String(), outputPath)
}

const scriptTemplate = `
#!/usr/bin/env bash
set -e

gcloud compute instances list
gcloud compute instances create {{.LogPrefix}}-{{.DBName}}-1 --custom-cpu=8 --custom-memory=16 --image="ubuntu-15-10" --boot-disk-size=50 --boot-disk-type="pd-ssd" --local-ssd interface=SCSI --zone us-central1-a
gcloud compute instances create {{.LogPrefix}}-{{.DBName}}-2 --custom-cpu=8 --custom-memory=16 --image="ubuntu-15-10" --boot-disk-size=50 --boot-disk-type="pd-ssd" --local-ssd interface=SCSI --zone us-central1-a
gcloud compute instances create {{.LogPrefix}}-{{.DBName}}-3 --custom-cpu=8 --custom-memory=16 --image="ubuntu-15-10" --boot-disk-size=50 --boot-disk-type="pd-ssd" --local-ssd interface=SCSI --zone us-central1-a
gcloud compute instances create {{.LogPrefix}}-{{.DBName}}-tester --custom-cpu=16 --custom-memory=30 --image="ubuntu-15-10" --boot-disk-size=50 --boot-disk-type="pd-ssd" --zone us-central1-a
gcloud compute instances list

gcloud compute ssh {{.LogPrefix}}-{{.DBName}}-1
gcloud compute ssh {{.LogPrefix}}-{{.DBName}}-2
gcloud compute ssh {{.LogPrefix}}-{{.DBName}}-3
gcloud compute ssh {{.LogPrefix}}-{{.DBName}}-tester


#########
# agent #
#########
GO_VERSION="1.6" && cd /usr/local && sudo rm -rf ./go && sudo curl -s https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | sudo tar -v -C /usr/local/ -xz && cd $HOME;
echo "export GOPATH=$(echo $HOME)/go" >> $HOME/.bashrc
PATH_VAR=$PATH":/usr/local/go/bin:$(echo $HOME)/go/bin"
echo "export PATH=$(echo $PATH_VAR)" >> $HOME/.bashrc
export GOPATH=$(echo $HOME)/go
PATH_VAR=$PATH":/usr/local/go/bin:$(echo $HOME)/go/bin"
export PATH=$(echo $PATH_VAR)
go get -v -u -f github.com/coreos/dbtester
curl https://storage.googleapis.com/etcd/dbtester_agent.sh | bash -s f /mnt/ssd0

cd /mnt/ssd0
ls /mnt/ssd0
cat /mnt/ssd0/agent.log


##########
# tester #
##########
ulimit -n 3000
ulimit -n
GO_VERSION="1.6" && cd /usr/local && sudo rm -rf ./go && sudo curl -s https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | sudo tar -v -C /usr/local/ -xz && cd $HOME;
echo "export GOPATH=$(echo $HOME)/go" >> $HOME/.bashrc
PATH_VAR=$PATH":/usr/local/go/bin:$(echo $HOME)/go/bin"
echo "export PATH=$(echo $PATH_VAR)" >> $HOME/.bashrc
export GOPATH=$(echo $HOME)/go
PATH_VAR=$PATH":/usr/local/go/bin:$(echo $HOME)/go/bin"
export PATH=$(echo $PATH_VAR)
psn ps-kill --force -s dbtester
go get -v -u -f github.com/coreos/dbtester

# start test
AGENT_ENDPOINTS='___IP_ADDR_1___:3500,___IP_ADDR_2___:3500,___IP_ADDR_3___:3500'
DATABASE_ENDPOINTS='___IP_ADDR_1___:{{.ClientPort}},___IP_ADDR_2___:{{.ClientPort}},___IP_ADDR_3___:{{.ClientPort}}'

# start database
dbtester start --agent-endpoints=$(echo $AGENT_ENDPOINTS) --database={{.DBName}} --database-log-path=database.log --log-prefix={{.LogPrefix}}-{{.DBName}} --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}} --monitor-result-path=monitor.csv;

cat /mnt/ssd0/agent.log
cat /mnt/ssd0/database.log

# start benchmark
nohup dbtester bench --database={{.DBName}} --sample --no-histogram --csv-result-path={{.LogPrefix}}-{{.DBName}}-timeseries.csv --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}} --endpoints=$DATABASE_ENDPOINTS --conns={{.Conns}} --clients={{.Clients}} put --key-size=64 --val-size={{.ValSize}} --total={{.Total}} > {{.LogPrefix}}-{{.DBName}}-result.txt 2>&1 &

cat {{.LogPrefix}}-{{.DBName}}-result.txt

# benchmark done!
# stop database to trigger uploading in remote machines
dbtester stop --agent-endpoints=$(echo $AGENT_ENDPOINTS)

dbtester upload --from={{.LogPrefix}}-{{.DBName}}-timeseries.csv --to={{.LogPrefix}}-{{.DBName}}-timeseries.csv --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}


####################
# in case of panic #
####################
dbtester upload --from=/mnt/ssd0/agent.log --to={{.LogPrefix}}-{{.DBName}}-1-agent.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/database.log --to={{.LogPrefix}}-{{.DBName}}-1-database.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/monitor.csv --to={{.LogPrefix}}-{{.DBName}}-1-monitor.csv --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}

dbtester upload --from=/mnt/ssd0/agent.log --to={{.LogPrefix}}-{{.DBName}}-2-agent.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/database.log --to={{.LogPrefix}}-{{.DBName}}-2-database.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/monitor.csv --to={{.LogPrefix}}-{{.DBName}}-2-monitor.csv --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}

dbtester upload --from=/mnt/ssd0/agent.log --to={{.LogPrefix}}-{{.DBName}}-3-agent.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/database.log --to={{.LogPrefix}}-{{.DBName}}-3-database.log --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}
dbtester upload --from=/mnt/ssd0/monitor.csv --to={{.LogPrefix}}-{{.DBName}}-3-monitor.csv --google-cloud-project-name={{.ProjectName}} --key-path={{.KeyPath}} --bucket={{.BucketName}}

`

func toFile(txt, fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	if _, err := f.WriteString(txt); err != nil {
		return err
	}
	return nil
}
