# Metrics Synopsis: apple-reconciler

- **Collected:** 2026-09-22T08:39:30Z
- **Interval:** 35m
- **Window:** 2026-09-22T08:04:30Z to 2026-09-22T08:39:30Z
- **Note:** 76k hosts - 2 profiles configured

## Summary (35m averages)

```
Fleet Server:  CPU=56.51%  Mem=7.34%  Containers=40
Loadtest:      CPU=7.56%  Mem=10.63%  Containers=38
apns-mock:     CPU=6.59%  Mem=32.76%  Containers=2  (averaged across containers)
               PerTask CPU avg=6.59% max=9.99%   Mem avg=32.76% max=39.6%
               RX=25.13MB  TX=26.97MB  AbnormalStops=0  StartSpread=5.52min
apns Redis:    CPU=0.98% (max 1.75%)  Mem=0.9%  Conns=13.86  PendingItems=3.83  Evictions=0
               StringCmds=304772  PubSubCmds=151966  NetOut=108.24MB
RDS Writer:    CPU=24.25%  Connections=251.83  Deadlocks=0
RDS reader-1:  CPU=44.91%  Connections=213.17
RDS reader-2:  CPU=26.65%  Connections=186.69
Redis:         CPU=53.09%  Mem=3.61%

Top SQL (writer):
  #1 load=1.34  COMMIT
  #2 load=0.58  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 
  #3 load=0.41  SELECT `s` . `id` FROM `software` `s` LEFT JOIN `software_host_counts` `shc` ON 
  #4 load=0.03  DELETE FROM `host_software_installed_paths` WHERE `id` IN (...) 
  #5 load=0.03  INSERT INTO `host_seen_times` ( `host_id` , `seen_time` ) VALUES (...) /* , ... 

Top SQL (reader-1):
  #1 load=0.55  ( SELECT `id` , NAME , `instance` , `stats_type` , STATUS , `created_at` , `upda
  #2 load=0.4  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #3 load=0.38  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #4 load=0.32  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #5 load=0.24  SELECT `id` , `host_id` , `execution_id` , `script_id` , `created_at` FROM ( SEL

Top SQL (reader-2):
  #1 load=0.21  SELECT `q` . NAME , `q` . QUERY , `q` . `team_id` , `q` . `schedule_interval` , 
  #2 load=0.21  SELECT `hm` . `host_id` , `hm` . `enrolled` , `hm` . `server_url` , `hm` . `inst
  #3 load=0.2  SELECT DISTINCTROW `packs` . * FROM ( ( SELECT `p` . * FROM `packs` `p` JOIN `pa
  #4 load=0.14  SELECT `sc` . `cve` , `hs` . `host_id` FROM `software_cve` `sc` JOIN `host_softw
  #5 load=0.14  SELECT `h` . `team_id` AS `team_id` , ? AS `global_stats` , `combined_results` .
ALB:           Latency=0s  5xx=65  Requests=74068146  Traffic=142814.75MB
Fleet Errors:  Count=0.0
RDS Writer:    FreeMem=27.01GB  CacheHit=100%  Disk=N/A  Threads=2.78  IOPS=12.48%
               SelectLat=2.15ms  InsertLat=3.71ms
RDS reader-1: FreeMem=28.2GB  CacheHit=100%  ReplicaLag=7.03ms  Threads=3.7  IOPS=0.07%
               SelectLat=0.33ms
RDS reader-2: FreeMem=28.22GB  CacheHit=100%  ReplicaLag=5.86ms  Threads=2.06  IOPS=0.05%
               SelectLat=0.36ms
Redis:         Connections=843.89  Evictions=0  CacheHit=84.56%
Network:       RX=2383.23MB  TX=762.14MB
Containers:    Running=38  AbnormalStops=0  StartSpread=N/Amin
```

## Threshold Checks

```
  ✅ All metrics within expected thresholds
```
