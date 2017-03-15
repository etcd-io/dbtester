#!/usr/bin/env bash
set -e

<<COMMENT
ETCD_VER=v3.1.1

GOOGLE_URL=https://storage.googleapis.com/etcd
GITHUB_URL=https://github.com/coreos/etcd/releases/download

DOWNLOAD_URL=${GOOGLE_URL}

rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
rm -rf /tmp/test-etcd-${ETCD_VER} && mkdir -p /tmp/test-etcd-${ETCD_VER}

curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp/test-etcd-${ETCD_VER} --strip-components=1

sudo cp /tmp/test-etcd-${ETCD_VER}/etcd* $GOPATH/bin
COMMENT

GIT_PATH=github.com/coreos/etcd

USER_NAME=coreos
BRANCH_NAME=master

rm -rf ${GOPATH}/src/${GIT_PATH}
mkdir -p ${GOPATH}/src/github.com/coreos

git clone https://github.com/${USER_NAME}/etcd \
    --branch ${BRANCH_NAME} \
    ${GOPATH}/src/${GIT_PATH}

cd ${GOPATH}/src/${GIT_PATH}

git reset --hard 5856c8bce9778a12d79038fdb1f3fba9416bd297

./build

${GOPATH}/src/${GIT_PATH}/bin/etcd --version
${GOPATH}/src/${GIT_PATH}/bin/etcdctl --version

cp ${GOPATH}/src/${GIT_PATH}/bin/etcd ${GOPATH}/bin/etcd
sudo cp ${GOPATH}/src/${GIT_PATH}/bin/etcd /etcd

cp ${GOPATH}/src/${GIT_PATH}/bin/etcdctl ${GOPATH}/bin/etcdctl
sudo cp ${GOPATH}/src/${GIT_PATH}/bin/etcdctl /etcdctl

$GOPATH/bin/etcd --version
$GOPATH/bin/etcdctl --version
etcd --version
etcdctl --version
