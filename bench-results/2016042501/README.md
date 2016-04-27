

- Google Cloud Compute Engine
- 8 vCPUs + 16GB Memory + 50GB SSD
- 1 machine(client) of 16 vCPUs + 30GB Memory + 50GB SSD
- Ubuntu 15.10
- Go master branch on 2016-04-25
- etcd v3 (compress branch)



<br><br><hr>
##### Write 2M keys, 1000-client, 100-conn, 8-byte key, 256-byte value (etcd v3)

<img src="https://storage.googleapis.com/dbtester-results/2016042501/01-avg-latency-ms.svg" alt="2016042501/01-avg-latency-ms">

<img src="https://storage.googleapis.com/dbtester-results/2016042501/01-throughput.svg" alt="2016042501/01-throughput">

<img src="https://storage.googleapis.com/dbtester-results/2016042501/01-avg-cpu.svg" alt="2016042501/01-avg-cpu">

<img src="https://storage.googleapis.com/dbtester-results/2016042501/01-avg-memory.svg" alt="2016042501/01-avg-memory">



