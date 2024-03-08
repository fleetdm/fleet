# Load testing

The following document outlines the most recent results of a semi-annual load test of the Fleet server. 

These tests are conducted by the Fleet team, using [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf); a free and open source tool, to generate realistic traffic to the Fleet server.

This document reports the minimum resources for successfully running Fleet with 1,000 hosts and 150,000 hosts.

## Test parameters

The Fleet load tests are conducted with a Fleet server that contains 2 packs, with ~6 queries each, and 6 labels.

A test is deemed successful when the Fleet server is able to receive and make requests to the specified number of hosts without over utilizing the specified resources. In addition, a successful test must report that the Fleet server can run a live query against the specified number of hosts.

## Results

### 2,500 hosts

With the following infrastructure, 2,500 hosts successfully communicate with Fleet. The Fleet server is able to run live queries against all hosts.

| Fleet instances | CPU Units     | RAM           |
|-----------------|---------------|---------------|
| 1 Fargate task  | 512 CPU Units | 4GB of memory |

| &#8203; | Version                 | Instance type    |
|---------|-------------------------|------------------|
| Redis   | 6.x                     | cache.t4g.medium |
| MySQL   | 8.0.mysql_aurora.3.02.0 | db.t4g.small     |

### 150,000 hosts

With the infrastructure listed below, 150,000 hosts successfully communicate with Fleet. The Fleet server is able to run live queries against all hosts.

| Fleet instance   | CPU Units      | RAM           |
|------------------|----------------|---------------|
| 20 Fargate tasks | 1024 CPU units | 4GB of memory |

| &#8203; | Version                 | Instance type   |
|---------|-------------------------|-----------------|
| Redis   | 6.x                     | cache.m6g.large |
| MySQL   | 8.0.mysql_aurora.3.02.0 | db.r6g.4xlarge  |

In the above setup, the read replica was the same size as the writer node.

The above setup auto scaled based on CPU usage. After a while, the task count ended up at 25 instances even while live querying or adding a new label.

## How we are simulating osquery

The simulation is run by using [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf), a free and open source tool, to generate realistic traffic to the Fleet server.

The following command enrolls and simulates 150,000 hosts on Fleet:

```bash
go run cmd/osquery-perf/agent.go -enroll_secret <secret here> -host_count 150000 -server_url <server URL here> -node_key_file nodekeys
```

After the hosts have been enrolled, you can add `-only_already_enrolled` to make sure the node keys from the file are used and no enrollment happens. This resumes the execution of all the simulated hosts.

## Infrastructure setup

The deployment of Fleet was done through the loadtesting [terraform maintained in the repo](https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform) with the following command:

```bash
terraform apply -var tag=<your tag here>
```

Scaling differences were done by directly modifying the code and reapplying.

Infrastructure for the loadtest is provided in the loadtesting code (via an ECS Fargate service and an internal load balancer for cost savins). Each instance of the ECS service corresponds to 5000 hosts.
They are sized to be the smallest that Fargate allows, so it is still cost effective to run 30+ instances of the service.

## Limitations

The [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf) tool doesn't simulate all data that's included when a real device communicates to a Fleet instance. For example, system users and software inventory data are not yet simulated by osquery-perf.

<meta name="maintainedBy" value="lukeheath">
<meta name="description" value="This page outlines the most recent results of a semi-annual load test of the Fleet server.">
