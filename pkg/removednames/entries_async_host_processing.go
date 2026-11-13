package removednames

func init() { register(asyncHostProcessingEntries...) }

var asyncHostProcessingEntries = []Entry{
	serverConfigEntry("osquery.enable_async_host_processing", ""),
	serverConfigEntry("osquery.async_host_collect_interval", ""),
	serverConfigEntry("osquery.async_host_collect_max_jitter_percent", ""),
	serverConfigEntry("osquery.async_host_collect_lock_timeout", ""),
	serverConfigEntry("osquery.async_host_collect_log_stats_interval", ""),
	serverConfigEntry("osquery.async_host_insert_batch", ""),
	serverConfigEntry("osquery.async_host_delete_batch", ""),
	serverConfigEntry("osquery.async_host_update_batch", ""),
	serverConfigEntry("osquery.async_host_redis_pop_count", ""),
	serverConfigEntry("osquery.async_host_redis_scan_keys_count", ""),
}
