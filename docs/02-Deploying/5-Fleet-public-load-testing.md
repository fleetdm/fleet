# Baseline

## baseline set of data

Baseline setup: 6 custom labels, 6 policies, and 2 packs with ~6 queries each, and be able to live query all the hosts

## how were the osquery hosts simulated

using osquery perf

## setup

fleet instances:

1 Task
256 CPU units
512 MB of memory

amount of hosts: 1000k

redis: 
version: 5.0.6
type: cluster 1 primary, 2 replicas
instance: cache.t2.micro

mysql: 
version: 5.7.mysql_aurora.2.10.0
instance: db.t4g.medium

## Limitations of the test

osquery perf doesn't have host users and inventory support yet

# 150k hosts

fleet instances:

25 Task
1024 CPU units
2048 MB of memory

amount of hosts: 150000k

redis:
version: 5.0.6
type: cluster 1 primary, 2 replicas
instance: cache.t2.micro

mysql:
version: 5.7.mysql_aurora.2.10.0
instance: db.r5.4xlarge

---

async writes: cache.m5.2xlarge