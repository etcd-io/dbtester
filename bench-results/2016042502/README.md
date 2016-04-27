

- Google Cloud Compute Engine
- 8 vCPUs + 16GB Memory + 50GB SSD
- 1 machine(client) of 16 vCPUs + 30GB Memory + 50GB SSD
- Ubuntu 15.10
- Go master branch on 2016-04-25
- etcd v3 (compress branch)



<br><br><hr>
##### Write 1M keys, 1000-client, 100-conn, 64-byte key, 1024-byte value (etcd v3)

<img src="https://storage.googleapis.com/dbtester-results/2016042502/01-avg-latency-ms.svg" alt="2016042502/01-avg-latency-ms">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/01-throughput.svg" alt="2016042502/01-throughput">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/01-avg-cpu.svg" alt="2016042502/01-avg-cpu">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/01-avg-memory.svg" alt="2016042502/01-avg-memory">



<br><br><hr>
##### Write 1M keys, 1000-client, 100-conn, 48-byte key, 500-byte value (etcd v3)

<img src="https://storage.googleapis.com/dbtester-results/2016042502/02-avg-latency-ms.svg" alt="2016042502/02-avg-latency-ms">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/02-throughput.svg" alt="2016042502/02-throughput">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/02-avg-cpu.svg" alt="2016042502/02-avg-cpu">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/02-avg-memory.svg" alt="2016042502/02-avg-memory">



<br><br><hr>
##### Write 1M keys, 1000-client, 100-conn, 48-byte key, 800-byte value (etcd v3)

<img src="https://storage.googleapis.com/dbtester-results/2016042502/03-avg-latency-ms.svg" alt="2016042502/03-avg-latency-ms">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/03-throughput.svg" alt="2016042502/03-throughput">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/03-avg-cpu.svg" alt="2016042502/03-avg-cpu">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/03-avg-memory.svg" alt="2016042502/03-avg-memory">



<br><br><hr>
##### Write 500K keys, 1-client, 1-conn, 64-byte key, 1024-byte value (etcd v3)

<img src="https://storage.googleapis.com/dbtester-results/2016042502/04-avg-latency-ms.svg" alt="2016042502/04-avg-latency-ms">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/04-throughput.svg" alt="2016042502/04-throughput">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/04-avg-cpu.svg" alt="2016042502/04-avg-cpu">

<img src="https://storage.googleapis.com/dbtester-results/2016042502/04-avg-memory.svg" alt="2016042502/04-avg-memory">



