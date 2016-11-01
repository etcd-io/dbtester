This is an experimental project. Project/code is subject to change anytime.
This is mainly for comparing etcd with other databases. For etcd, we recommend
to just run [etcd benchmark tool](https://github.com/coreos/etcd/tree/master/tools/benchmark).

# dbtester

[![Build Status](https://img.shields.io/travis/coreos/dbtester.svg?style=flat-square)][cistat] [![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][dbtester-godoc]

Distributed database tester.

Please let us know or file an issue if:

- Need help with running this testing suite
- Questions about test results

We keep all logs at cloud storage:

- https://console.cloud.google.com/storage/browser/dbtester-results

Test results:

- Read 200 keys, 1-conn, 1-client, 8-byte key, 3MB value (etcd v3, compression): [2016052701](https://github.com/coreos/dbtester/tree/master/bench-results/2016052701/README.md)
- Write 2M keys, 1000-client (100-conn), 8-byte key, 256-byte value (Zookeeper v3.4.8, etcd v2, v3, Go, Consul v0.6.4): [2016051401](https://github.com/coreos/dbtester/tree/master/bench-results/2016051401/README.md)
- Snappy compression experiment (etcd v3, 1.3MB text value READ): [2016051001](https://github.com/coreos/dbtester/tree/master/bench-results/2016051001/README.md)
- Snappy compression experiment (etcd v3, 0.3MB text value READ): [2016050901](https://github.com/coreos/dbtester/tree/master/bench-results/2016050901/README.md)
- Write 2M keys, 1000-client, 1000-conn, 8-byte same key, 256-byte value (Zookeeper v3.4.8): [2016050504](https://github.com/coreos/dbtester/tree/master/bench-results/2016050504/README.md)
- Write 1M keys, 1000-client, 1000-conn, 8-byte same key, 256-byte value (Zookeeper v3.4.8): [2016050503](https://github.com/coreos/dbtester/tree/master/bench-results/2016050503/README.md)
- Write 600K keys, 1000-client, 1000-conn, 8-byte same key, 256-byte value (Zookeeper v3.4.8, Consul v0.6.4): [2016050502](https://github.com/coreos/dbtester/tree/master/bench-results/2016050502/README.md)
- Write 600K keys, 1000-client, 1000-conn, 8-byte key, 256-byte value (etcd v2): [2016050501](https://github.com/coreos/dbtester/tree/master/bench-results/2016050501/README.md)
- Write 200K, 400K, 600K keys, 1000-client, 1000-conn, 8-byte key, 256-byte value (etcd v2): [2016050301](https://github.com/coreos/dbtester/tree/master/bench-results/2016050301/README.md)
- Snappy compression experiment (etcd v3, 0.3MB text value WRITE): [2016050101](https://github.com/coreos/dbtester/tree/master/bench-results/2016050101/README.md)
- Snappy, Lz4 compression experiment (etcd v3, 1.0MB text value WRITE): [2016043002](https://github.com/coreos/dbtester/tree/master/bench-results/2016043002/README.md)
- MVCC patch (slice capacity), Write 2M keys, 1000-client, 100-conn, 8-byte key, 256-byte value (etcd v3): [2016043001](https://github.com/coreos/dbtester/tree/master/bench-results/2016043001/README.md)
- Snappy compression experiment (etcd v3, 1.0MB value WRITE): [2016042502](https://github.com/coreos/dbtester/tree/master/bench-results/2016042502/README.md)
- Snappy, cgzip, gzip compression experiment (etcd v3, 256-byte value WRITE): [2016042501](https://github.com/coreos/dbtester/tree/master/bench-results/2016042501/README.md)
- Write 200K keys, 1000-client, 8-byte key, 256-byte value (etcd v2): [2016041801](https://github.com/coreos/dbtester/tree/master/bench-results/2016041801/README.md)
- Write 2M keys, 1000-client (etcdv3 100-conns), 8-byte key, 256-byte value (etcd v2, etcd v3, Zookeeper v3.4.8, Consul v0.6.4): [2016041601](https://github.com/coreos/dbtester/tree/master/bench-results/2016041601/README.md)
- SnapCount experiment, Write 2M keys, 1,000 clients, 8-byte key, 256-byte value (Zookeeper v3.4.8): [2016041502](https://github.com/coreos/dbtester/tree/master/bench-results/2016041502/README.md)
- SnapCount experiment, Write 2M keys, 1,000 clients, 8-byte key, 256-byte value (etcd v3, Zookeeper v3.4.8): [2016041501](https://github.com/coreos/dbtester/tree/master/bench-results/2016041501/README.md)
- SnapCount experiment, Write 500K keys, 1 client, 32-byte key, 500-byte value (etcd v3, Zookeeper v3.4.8): [2016041401](https://github.com/coreos/dbtester/tree/master/bench-results/2016041401/README.md)
- Write 1M keys, 500 clients(etcd 50 conns), key 32 bytes, value 500 bytes (etcd v3, Zookeeper v3.4.8): [2016041203](https://github.com/coreos/dbtester/tree/master/bench-results/2016041203/README.md)
- Write 1M keys, 1 client, key 64 bytes, value 256 bytes (etcd v3, Zookeeper v3.4.8): [2016041202](https://github.com/coreos/dbtester/tree/master/bench-results/2016041202/README.md)
- Write 300K keys, 1 client, key 64 bytes, value 256 bytes (etcd v3, Zookeeper v3.4.8): [2016041201](https://github.com/coreos/dbtester/tree/master/bench-results/2016041201/README.md)

[cistat]: https://travis-ci.org/coreos/dbtester
[dbtester-godoc]: https://godoc.org/github.com/coreos/dbtester
