# Metrics Synopsis: apple-reconciler

- **Collected:** 2026-09-22T09:29:55Z
- **Interval:** 47m
- **Window:** 2026-09-22T08:42:55Z to 2026-09-22T09:29:55Z
- **Note:** 76k hosts - 4 extra profiles configured (total 6)

## Summary (47m averages)

```
Fleet Server:  CPU=56.66%  Mem=9.24%  Containers=40
Loadtest:      CPU=7.69%  Mem=11.04%  Containers=38
apns-mock:     CPU=6.76%  Mem=33.25%  Containers=2  (averaged across containers)
               PerTask CPU avg=6.76% max=14.16%   Mem avg=33.25% max=39.6%
               RX=35.96MB  TX=37.95MB  AbnormalStops=0  StartSpread=5.51min
apns Redis:    CPU=1.28% (max 3.15%)  Mem=0.9%  Conns=15.36  PendingItems=109.21  Evictions=0
               StringCmds=589936  PubSubCmds=294416  NetOut=198.58MB
RDS Writer:    CPU=20.76%  Connections=342.89  Deadlocks=0
RDS reader-1:  CPU=49.31%  Connections=225.04
RDS reader-2:  CPU=20.52%  Connections=177.32
Redis:         CPU=52.02%  Mem=3.6%

Top SQL (writer):
  #1 load=2.22  COMMIT
  #2 load=0.76  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #3 load=0.09  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #4 load=0.05  SELECT `c` . `command_uuid` , `c` . `request_type` , `c` . `command` , `c` . `su
  #5 load=0.05  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 

Top SQL (reader-1):
  #1 load=0.47  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #2 load=0.45  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #3 load=0.39  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.28  SELECT `id` , `host_id` , `execution_id` , `script_id` , `created_at` FROM ( SEL
  #5 load=0.28  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 

Top SQL (reader-2):
  #1 load=0.17  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #2 load=0.17  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #3 load=0.13  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.1  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #5 load=0.1  SELECT `id` , `host_id` , `execution_id` , `script_id` , `created_at` FROM ( SEL
ALB:           Latency=0s  5xx=12453  Requests=99715465  Traffic=192662.24MB
Fleet Errors:  Count=0.0
apns-mock Errors: Count=2.0
RDS Writer:    FreeMem=27.02GB  CacheHit=100%  Disk=29.66GB  Threads=3.64  IOPS=12.1%
               SelectLat=0.33ms  InsertLat=3.56ms
RDS reader-1: FreeMem=28.17GB  CacheHit=100%  ReplicaLag=8.64ms  Threads=3.9  IOPS=0.07%
               SelectLat=0.34ms
RDS reader-2: FreeMem=28.23GB  CacheHit=100%  ReplicaLag=6.43ms  Threads=1.28  IOPS=0.06%
               SelectLat=0.32ms
Redis:         Connections=441.55  Evictions=0  CacheHit=84.42%
Network:       RX=3210.83MB  TX=1033.4MB
Containers:    Running=38  AbnormalStops=0  StartSpread=N/Amin
```

## Threshold Checks

```
  🔴 ALERT: apns-mock Errors = 2.0 (expected 0)

  ⚠ 1 alert(s) detected
```
