# Metrics Synopsis: apple-reconciler

- **Collected:** 2026-09-22T08:04:13Z
- **Interval:** 3h
- **Window:** 2026-09-22T05:04:13Z to 2026-09-22T08:04:13Z
- **Note:** 76k hosts initial baseline

## Summary (3h averages)

```
Fleet Server:  CPU=55.89%  Mem=6.53%  Containers=40
Loadtest:      CPU=7.3%  Mem=10.49%  Containers=38
apns-mock:     CPU=4.86%  Mem=31.29%  Containers=2  (averaged across containers)
               PerTask CPU avg=4.86% max=7.54%   Mem avg=31.29% max=40.04%
               RX=112.98MB  TX=126.57MB  AbnormalStops=0  StartSpread=5.51min
apns Redis:    CPU=0.28% (max 0.65%)  Mem=0.89%  Conns=8.73  PendingItems=3  Evictions=0
               StringCmds=460320  PubSubCmds=6  NetOut=116.31MB
RDS Writer:    CPU=18.94%  Connections=265.14  Deadlocks=0
RDS reader-1:  CPU=38.06%  Connections=203.99
RDS reader-2:  CPU=32.2%  Connections=194.36
Redis:         CPU=51.94%  Mem=3.6%

Top SQL (writer):
  #1 load=0.74  COMMIT
  #2 load=0.68  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #3 load=0.09  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #4 load=0.09  SELECT `s` . `id` FROM `software` `s` LEFT JOIN `software_host_counts` `shc` ON 
  #5 load=0.04  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 

Top SQL (reader-1):
  #1 load=0.34  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #2 load=0.33  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #3 load=0.28  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.21  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #5 load=0.2  SELECT `h` . `id` AS `id` , `h` . `uuid` AS `uuid` , `h` . `team_id` AS `team_id

Top SQL (reader-2):
  #1 load=0.28  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #2 load=0.28  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #3 load=0.23  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.22  SELECT `h` . `id` AS `id` , `h` . `uuid` AS `uuid` , `h` . `team_id` AS `team_id
  #5 load=0.17  SELECT `id` , `host_id` , `execution_id` , `script_id` , `created_at` FROM ( SEL
ALB:           Latency=0s  5xx=15  Requests=378513566  Traffic=725387.78MB
Fleet Errors:  Count=0.0
RDS Writer:    FreeMem=27.02GB  CacheHit=100%  Disk=29.15GB  Threads=2  IOPS=11.8%
               SelectLat=1.5ms  InsertLat=4.46ms
RDS reader-1: FreeMem=28.2GB  CacheHit=100%  ReplicaLag=7.32ms  Threads=2.81  IOPS=0.01%
               SelectLat=0.36ms
RDS reader-2: FreeMem=28.2GB  CacheHit=100%  ReplicaLag=6.08ms  Threads=2.45  IOPS=0%
               SelectLat=0.36ms
Redis:         Connections=455.78  Evictions=0  CacheHit=84.93%
Network:       RX=12030.96MB  TX=3812.5MB
Containers:    Running=38  AbnormalStops=0  StartSpread=N/Amin
```

## Threshold Checks

```
  ✅ All metrics within expected thresholds
```
