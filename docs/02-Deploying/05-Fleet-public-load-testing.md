# Load testing

## Baseline Test

Baseline setup: 6 custom labels, 6 policies, and 2 packs with ~6 queries each, and be able to live query all the hosts.

## How we are simulating osquery

The simulation is run by using [osquery-perf](https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf) using the following command:

```bash
go run cmd/osquery-perf/agent.go -enroll_secret <secret here> -host_count 150000 -server_url <server URL here> -node_key_file nodekeys
```

After the hosts have been enrolled, you can simply add `-only_already_enrolled` to make sure the node keys from the file 
are used and no enrollment happens, virtually "resuming" the execution of all the simulated hosts.

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

## Bare minimum setup

Fleet instances:
- 1 Fargate Task
- 256 CPU units
- 512 MB of memory
- Amount of hosts: 1000

Redis: 
- Version: 5.0.6
- Instance type: cache.m5.large

Mysql:
- Version: 5.7.mysql_aurora.2.10.0
- Instance type: db.t4g.medium

With the above infrastructure, 1000 hosts were able to run and be live query without a problem.

## 150k hosts

Fleet instances:
- 25 Task
- 1024 CPU units
- 2048 MB of memory
- Amount of hosts: 150000k

Redis:
- Version: 5.0.6
- Instance: cache.m5.large

Mysql:
- Version: 5.7.mysql_aurora.2.10.0
- Instance: db.r5.4xlarge

The setup auto scaled based on CPU usage. After a while, the task count ended up in 25 instances even while live querying 
or adding a new label. 

## Limitations of the test

While osquery-perf simulates enough of osquery to be a good first step, it's not the smartest simulation as of the time of
this writing. Particularly, it doesn't simulate host users and software inventory yet.
