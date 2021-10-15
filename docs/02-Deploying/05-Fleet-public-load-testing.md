# Load testing

The following document outlines the most recent results of a semi-annual load test on the Fleet server conducted by the Fleet team. 

Fleet uses [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf), a free and open source tool, to generate realistic traffic to the Fleet server.

A test is deemed successful when the Fleet server is able to receive and make requests to the specified number of hosts without over utilizing the specified resources. In addition, a successful test must report that the Fleet server can run a live query against the specified number of hosts.

This document reports the minimum resources for successfully running Fleet with 1,000 hosts and 150,000 hosts. 

## Test parameters

The Fleet load tests are conducted with a Fleet server that contains 2 packs, with ~6 queries each, and 6 labels.

## How we are simulating osquery

The simulation is run by using [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf).

The following command enrolls and simulates 150,000 hosts on Fleet:

```bash
go run cmd/osquery-perf/agent.go -enroll_secret <secret here> -host_count 150000 -server_url <server URL here> -node_key_file nodekeys
```

After the hosts have been enrolled, you can add `-only_already_enrolled` to make sure the node keys from the file are used and no enrollment happens. This resumes the execution of all the simulated hosts.

## Infrastructure setup

The deployment of Fleet was done through the example [terraform provided in the repo](https://github.com/fleetdm/fleet/tree/main/tools/terraform) with the following command:

```bash
terraform apply \ 
  -var domain_fleetctl=<your domain here> \
  -var domain_fleetdm=<alternative domain here> \ 
  -var s3_bucket=<log bucket name> \
  -var fleet_image="fleetdm/fleet:<tag targeted>" \
  -var vulnerabilities_path="" \
  -var fleet_max_capacity=100 \ 
  -var fleet_min_capacity=5
```

## Results

### 1,000 hosts

Fleet instances:
- 1 Fargate Task
- 256 CPU units
- 512 MB of memory

Redis: 
- Version: 5.0.6
- Instance type: cache.m5.large

MySQL:
- Version: 5.7.mysql_aurora.2.10.0
- Instance type: db.t4g.medium

With the above infrastructure, 1,000 hosts successfully communicate with Fleet. The Fleet server is able to run live queries against all hosts.

### 150,000 hosts

Fleet instances:
- 25 Task
- 1024 CPU units
- 2048 MB of memory

Redis:
- Version: 5.0.6
- Instance: cache.m5.large

MySQL:
- Version: 5.7.mysql_aurora.2.10.0
- Instance: db.r5.4xlarge

With the above infrastructure, 150,000 hosts successfully communicate with Fleet. The Fleet server is able to run live queries against all hosts.

The above setup auto scaled based on CPU usage. After a while, the task count ended up in 25 instances even while live querying or adding a new label. 

## Limitations

The [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf) tool doesn't simulate all data that's included when a real device communicates to a Fleet instance. For example, system users and software inventory data are not yet simulated by osquery-perf.
