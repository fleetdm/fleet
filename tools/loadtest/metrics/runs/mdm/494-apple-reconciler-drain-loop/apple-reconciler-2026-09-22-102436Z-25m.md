# Metrics Synopsis: apple-reconciler

- **Collected:** 2026-09-22T10:24:36Z
- **Interval:** 25m
- **Window:** 2026-09-22T09:59:36Z to 2026-09-22T10:24:36Z
- **Note:** 76k hosts - 6 profiles removed

## Summary (25m averages)

```
Fleet Server:  CPU=57.69%  Mem=10.38%  Containers=40
Loadtest:      CPU=8.18%  Mem=11.34%  Containers=38
apns-mock:     CPU=8.14%  Mem=35.15%  Containers=2  (averaged across containers)
               PerTask CPU avg=8.14% max=13.17%   Mem avg=35.15% max=39.99%
               RX=25.27MB  TX=24.81MB  AbnormalStops=0  StartSpread=5.52min
apns Redis:    CPU=2.59% (max 3.58%)  Mem=0.89%  Conns=15.48  PendingItems=85.2  Evictions=0
               StringCmds=858802  PubSubCmds=429422  NetOut=259.19MB
RDS Writer:    CPU=28.27%  Connections=361.92  Deadlocks=0.32
RDS reader-1:  CPU=31.04%  Connections=196.24
RDS reader-2:  CPU=43.53%  Connections=205.04
Redis:         CPU=52.4%  Mem=3.6%

Top SQL (writer):
  #1 load=1.93  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #2 load=0.93  COMMIT
  #3 load=0.47  SELECT `s` . `id` FROM `software` `s` LEFT JOIN `software_host_counts` `shc` ON 
  #4 load=0.28  DELETE FROM `host_mdm_apple_profiles` WHERE `host_uuid` = ? AND `command_uuid` =
  #5 load=0.27  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 

Top SQL (reader-1):
  #1 load=0.28  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #2 load=0.25  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #3 load=0.22  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.15  SELECT `id` , `host_id` , `execution_id` , `script_id` , `created_at` FROM ( SEL
  #5 load=0.15  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 

Top SQL (reader-2):
  #1 load=0.37  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #2 load=0.36  SELECT `h` . `team_id` AS `team_id` , ? AS `global_stats` , `combined_results` .
  #3 load=0.32  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.31  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #5 load=0.22  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
ALB:           Latency=0s  5xx=25351  Requests=53848890  Traffic=103626.9MB
Fleet Errors:  Count=0.0
RDS Writer:    FreeMem=26.85GB  CacheHit=100%  Disk=30.44GB  Threads=5.2  IOPS=12.07%
               SelectLat=0.67ms  InsertLat=6.66ms
RDS reader-1: FreeMem=28.24GB  CacheHit=99.99%  ReplicaLag=7.36ms  Threads=1.92  IOPS=0.13%
               SelectLat=0.31ms
RDS reader-2: FreeMem=28.08GB  CacheHit=100%  ReplicaLag=7.48ms  Threads=3.35  IOPS=0.14%
               SelectLat=0.36ms
Redis:         Connections=482.36  Evictions=0  CacheHit=83.7%
Network:       RX=1723.34MB  TX=564.01MB
Containers:    Running=38  AbnormalStops=0  StartSpread=N/Amin
```

## Threshold Checks

```
  🔴 ALERT: RDS Writer Deadlocks = 0.32 (expected 0)

  ⚠ 1 alert(s) detected
```
