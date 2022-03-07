package schedule

// func doAsyncCleanupDistributedQueryCampaigns(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, config config.FleetConfig, instanceID string) (interface{}, error) {
// 	stats := make(map[string]interface{})
// 	if locked, err := ds.Lock(ctx, lockKeyLeader, instanceID, time.Hour); err != nil || !locked {
// 		level.Debug(logger).Log("leader", "Not the leader. Skipping...")
// 		stats["leader"] = "Not the leader. Skipping..."
// 		return stats, nil
// 	}
// 	expired, err := ds.CleanupDistributedQueryCampaigns(ctx, time.Now())
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["expired"] = expired
// 	return stats, nil
// }
