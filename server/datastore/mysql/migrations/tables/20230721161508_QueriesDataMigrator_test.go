package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230721161508(t *testing.T) {
	db := applyUpToPrev(t)

	dataStmts := `
		INSERT INTO users VALUES
			(1,'2023-07-21 20:32:32','2023-07-21 20:32:32',_binary '$2a$12$n6hwsD7OU2bAXX94551DQOBcNNhfsEPS3Y6JEuLDjsLNvry3lgJjy','0fF81xRQIriYzm5fdXouk3V3tRwsZJhV','admin','admin@email.com',0,'','',0,'admin',0),
			(2,'2023-07-21 20:33:13','2023-07-21 20:35:26',_binary '$2a$12$YxPPOd5TOmYhDlH5CfGIfuxBe4GJ78gbwvtxoBHTTw.symxpVcEZS','JPDLcBcv4j1QwIU+rHoRWBt3HVJC8hnf','User 1','user1@email.com',0,'','',0,NULL,0),
			(3,'2023-07-21 20:33:31','2023-07-21 20:36:42',_binary '$2a$12$u3kuHl44jMojsols1NayLu0pPBwZvnWH6j6ZuDk6HsN4r0jgg7BRu','MoWlTEHH9zR7blcJ0l7/1c4EMnkh/dxq','User2','user2@email.com',0,'','',0,NULL,0);

		INSERT INTO teams VALUES
			(1,'2023-07-21 20:32:42','Team 1','','{\"mdm\": {\"macos_setup\": {\"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null, \"enable_disk_encryption\": false}}, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": true}, \"integrations\": {\"jira\": null, \"zendesk\": null}, \"agent_options\": {\"config\": {\"options\": {\"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"webhook_settings\": {\"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}}'),
			(2,'2023-07-21 20:32:47','Team 2','','{\"mdm\": {\"macos_setup\": {\"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null, \"enable_disk_encryption\": false}}, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": true}, \"integrations\": {\"jira\": null, \"zendesk\": null}, \"agent_options\": {\"config\": {\"options\": {\"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"webhook_settings\": {\"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}}');

		INSERT INTO user_teams (user_id, team_id, role) VALUES
			(2,1,'admin'),
			(2,2,'admin'),
			(3,2,'admin'),
			(3,1,'observer');

		INSERT INTO packs (id, created_at, updated_at, disabled, name, description, platform, pack_type) VALUES
			(1,'2023-07-21 20:33:49','2023-07-21 20:33:49',0,'Global','Global pack','','global'),
			(2,'2023-07-21 20:34:34','2023-07-21 20:34:34',0,'performance-metrics','','',NULL),
			(3,'2023-07-21 20:36:03','2023-07-21 20:36:03',0,'Team: Team 1','Schedule additional queries for all hosts assigned to this team.','','team-1'),
			(4,'2023-07-21 20:36:45','2023-07-21 20:36:45',0,'Team: Team 2','Schedule additional queries for all hosts assigned to this team.','','team-2');

		INSERT INTO queries (id, created_at, updated_at, saved, name, description, query, author_id, observer_can_run, team_id, team_id_char, platform, min_osquery_version, schedule_interval, automations_enabled, logging_type) VALUES
			(1,'2023-07-21 20:33:47','2023-07-21 20:33:47',1,'Admin Global Query','Admin desc','SELECT * FROM osquery_info;',1,1,NULL,'','','',0,0,''),
			(2,'2023-07-21 20:34:34','2023-07-21 20:34:34',1,'per_query_perf','Records the CPU time and memory usage for each individual query. Helpful for identifying queries that may impact performance.','SELECT name, interval, executions, output_size, wall_time, (user_time/executions) AS avg_user_time, (system_time/executions) AS avg_system_time, average_memory FROM osquery_schedule;',1,0,NULL,'','','',0,0,''),
			(3,'2023-07-21 20:34:34','2023-07-21 20:34:34',1,'runtime_perf','Track the amount of CPU time used by osquery.','SELECT ov.version AS os_version, ov.platform AS os_platform, ov.codename AS os_codename, i.*, p.resident_size, p.user_time, p.system_time, time.minutes AS counter, db.db_size_mb AS database_size FROM osquery_info i, os_version ov, processes p, time, (SELECT (sum(size) / 1024) / 1024.0 AS db_size_mb FROM (SELECT value FROM osquery_flags WHERE name = \'database_path\' LIMIT 1) flags, file WHERE path LIKE flags.value || \'%%\' AND type = \'regular\') db WHERE p.pid = i.pid;',1,0,NULL,'','','',0,0,''),
			(4,'2023-07-21 20:34:34','2023-07-21 20:34:34',1,'endpoint_security_tool_perf','Track the percentage of total CPU time utilized by $endpoint_security_tool','SELECT ((tool_time*100)/(SUM(system_time) + SUM(user_time))) AS pct FROM processes, (SELECT (SUM(processes.system_time)+SUM(processes.user_time)) AS tool_time FROM processes WHERE name=\'endpoint_security_tool\');',1,0,NULL,'','','',0,0,''),
			(5,'2023-07-21 20:34:34','2023-07-21 20:34:34',1,'backup_tool_perf','Track the percentage of total CPU time utilized by $backup_tool','SELECT ((backuptool_time*100)/(SUM(system_time) + SUM(user_time))) AS pct FROM processes, (SELECT (SUM(processes.system_time)+SUM(processes.user_time)) AS backuptool_time FROM processes WHERE name=\'backup_tool\');',1,0,NULL,'','','',0,0,''),
			(6,'2023-07-21 20:35:37','2023-07-21 20:35:37',1,'User 1 Query','User 1 Query Desc','SELECT * FROM osquery_info;',2,0,NULL,'','','',0,0,''),
			(7,'2023-07-21 20:36:02','2023-07-21 20:36:02',1,'User 1 Query 2','','SELECT * FROM osquery_info;',2,1,NULL,'','','',0,0,''),
			(8,'2023-07-21 20:37:01','2023-07-21 20:37:01',1,'User 2 Query','Some desc','SELECT * FROM osquery_info;',3,1,NULL,'','','',0,0,''),
			(9,'2023-07-21 20:37:01','2023-07-21 20:37:01',1,'User 2 Query 2','Some desc','SELECT * FROM osquery_info;',3,1,NULL,'','','',0,0,'');

		INSERT INTO scheduled_queries VALUES
			-- Global pack
			(1,'2023-07-21 20:33:54','2023-07-21 20:33:54',1,1,86400,1,0,'','',NULL,'Admin Global Query','Admin Global Query','',NULL,''),
			(2,'2023-07-21 20:34:00','2023-07-21 20:34:00',1,1,3600,1,0,'','',NULL,'Admin Global Query','Admin Global Query-1','',NULL,''),

			-- 2017 pack
			(3,'2023-07-21 20:34:34','2023-07-21 20:34:34',2,NULL,1800,1,NULL,NULL,NULL,NULL,'per_query_perf','per_query_perf','Records the CPU time and memory usage for each individual query. Helpful for identifying queries that may impact performance.',NULL,''),
			(4,'2023-07-21 20:34:34','2023-07-21 20:34:34',2,NULL,1800,1,NULL,NULL,NULL,NULL,'runtime_perf','runtime_perf','Track the amount of CPU time used by osquery.',NULL,''),
			(5,'2023-07-21 20:34:34','2023-07-21 20:34:34',2,NULL,1800,1,NULL,NULL,NULL,NULL,'endpoint_security_tool_perf','endpoint_security_tool_perf','Track the percentage of total CPU time utilized by $endpoint_security_tool',NULL,''),
			(6,'2023-07-21 20:34:34','2023-07-21 20:34:34',2,NULL,1800,1,NULL,NULL,NULL,NULL,'backup_tool_perf','backup_tool_perf','Track the percentage of total CPU time utilized by $backup_tool',NULL,''),

			-- Global pack
			(7,'2023-07-21 20:34:46','2023-07-21 20:34:46',1,2,86400,1,0,'','',NULL,'per_query_perf','per_query_perf','',NULL,''),
			-- NULL platform
			(8,'2023-07-21 20:34:51','2023-07-21 20:34:51',1,2,86400,1,0,NULL,'',NULL,'per_query_perf','per_query_perf-1','',NULL,''),

			-- Team-1 pack
			(9,'2023-07-21 20:36:08','2023-07-21 20:36:08',3,6,86400,1,0,'','',NULL,'User 1 Query','User 1 Query','',NULL,''),
			-- NULL version
			(10,'2023-07-21 20:36:13','2023-07-21 20:36:13',3,6,86400,1,0,'',NULL,NULL,'User 1 Query','User 1 Query-1','',NULL,''),
			-- NULL platform
			(11,'2023-07-21 20:36:25','2023-07-21 20:36:25',3,2,86400,1,0,NULL,'',NULL,'per_query_perf','per_query_perf','',NULL,''),

			-- Team-2 pack
			(12,'2023-07-21 20:36:50','2023-07-21 20:36:50',4,5,86400,1,0,'','',NULL,'backup_tool_perf','backup_tool_perf','',NULL,'');
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// 'User 2 Query' is non-scheduled and was created by user#3, so it should exists in both the
	// global team and on team#2
	stmt := "SELECT description, query, author_id, saved, observer_can_run, team_id, team_id_char FROM queries WHERE name = ?"
	rows, err := db.Query(stmt, "User 2 Query")
	require.NoError(t, err)
	defer rows.Close()

	var nRows int
	var teamIDs []uint
	var teamIDStrs []string

	for rows.Next() {
		nRows += 1
		var teamIDStr string
		query := _20230719152138_Query{}
		err := rows.Scan(
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.TeamID,
			&teamIDStr,
		)
		require.NoError(t, err)
		require.Equal(t, query.Description, "Some desc")
		require.Equal(t, query.Query, "SELECT * FROM osquery_info;")
		require.Equal(t, *query.AuthorID, uint(3))
		require.Equal(t, query.Saved, true)
		require.Equal(t, query.ObserverCanRun, true)

		teamIDStrs = append(teamIDStrs, teamIDStr)
		if query.TeamID != nil {
			teamIDs = append(teamIDs, *query.TeamID)
		}
	}
	require.NoError(t, rows.Err())
	require.Equal(t, nRows, 2)
	require.ElementsMatch(t, teamIDStrs, []string{"", "2"})
	require.Contains(t, teamIDs, uint(2))

	// The global pack has 4 different schedules two targeting 'Admin Global Query' and the other
	// two targeting 'per_query_perf' so I expect to see 6 queries here:
	// 'Admin Global Query - 1 - $timestamp' <- For schedule with id 1
	// 'Admin Global Query - 2 - $timestamp' <- For schedule with id 2
	// 'per_query_perf' <- Original (kept because is referenced by an 2017 pack)
	// 'per_query_perf - 7 - $timestamp' <- For schedule with id 7
	// 'per_query_perf - 8 - $timestamp' <- For schedule with id 8
	stmt = `SELECT
				name,
				description,
				query,
				author_id,
				saved,
				observer_can_run,
				platform,
				min_osquery_version,
				schedule_interval,
				logging_type,
				automations_enabled
			FROM queries WHERE name LIKE ? AND team_id IS NULL
	`

	rows, err = db.Query(stmt, "Admin Global Query%")
	require.NoError(t, err)
	defer rows.Close()

	nRows = 0
	var names []string
	var scheduleIntervals []uint
	var automationsEnabled []bool
	var loggingTypes []string

	for rows.Next() {
		nRows += 1
		query := _20230719152138_Query{}
		err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.Platform,
			&query.MinOsqueryVersion,
			&query.ScheduleInterval,
			&query.LoggingType,
			&query.AutomationsEnabled,
		)
		require.NoError(t, err)

		names = append(names, query.Name)
		scheduleIntervals = append(scheduleIntervals, query.ScheduleInterval)
		automationsEnabled = append(automationsEnabled, query.AutomationsEnabled)
		loggingTypes = append(loggingTypes, query.LoggingType)

		require.Equal(t, query.Description, "Admin desc")
		require.Equal(t, query.Query, "SELECT * FROM osquery_info;")
		require.Equal(t, *query.AuthorID, uint(1))
		require.Equal(t, query.Saved, true)
		require.Equal(t, query.ObserverCanRun, true)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, names, []string{"Admin Global Query - 1 - Jul 21 20:33:54.000", "Admin Global Query - 2 - Jul 21 20:34:00.000"})
	require.ElementsMatch(t, scheduleIntervals, []uint{3600, 86400})
	require.ElementsMatch(t, automationsEnabled, []bool{true, true})
	require.ElementsMatch(t, loggingTypes, []string{"snapshot", "snapshot"})
	require.Equal(t, nRows, 2)

	rows, err = db.Query(stmt, "per_query_perf%")
	require.NoError(t, err)
	defer rows.Close()

	nRows = 0
	names = []string{}
	scheduleIntervals = []uint{}
	automationsEnabled = []bool{}
	loggingTypes = []string{}

	for rows.Next() {
		nRows += 1
		query := _20230719152138_Query{}
		err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.Platform,
			&query.MinOsqueryVersion,
			&query.ScheduleInterval,
			&query.LoggingType,
			&query.AutomationsEnabled,
		)
		require.NoError(t, err)

		names = append(names, query.Name)
		scheduleIntervals = append(scheduleIntervals, query.ScheduleInterval)
		automationsEnabled = append(automationsEnabled, query.AutomationsEnabled)
		loggingTypes = append(loggingTypes, query.LoggingType)

		require.Equal(t, query.Description, "Records the CPU time and memory usage for each individual query. Helpful for identifying queries that may impact performance.")
		require.Equal(t, query.Query, "SELECT name, interval, executions, output_size, wall_time, (user_time/executions) AS avg_user_time, (system_time/executions) AS avg_system_time, average_memory FROM osquery_schedule;")
		require.Equal(t, *query.AuthorID, uint(1))
		require.Equal(t, query.Saved, true)
		require.Equal(t, query.ObserverCanRun, false)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, names, []string{"per_query_perf", "per_query_perf - 7 - Jul 21 20:34:46.000", "per_query_perf - 8 - Jul 21 20:34:51.000"})
	require.ElementsMatch(t, scheduleIntervals, []uint{0, 86400, 86400})
	require.ElementsMatch(t, automationsEnabled, []bool{false, true, true})
	require.ElementsMatch(t, loggingTypes, []string{"", "snapshot", "snapshot"})
	require.Equal(t, nRows, 3)

	// We have two team packs (Team-1, Team-2)
	// For Team-1, we have three schedules, two of them reference 'User 1 Query', the last one
	// 'per_query_perf', so I expect to see five different queries on team#1:
	//   - 'User 1 Query - 9 - $timestamp' for schedule#9
	//   - 'User 1 Query - 10 - $timestamp' for schedule#10
	//   - 'per_query_perf - 11 - $timestamp' for schedule#11
	// For Team-2, we only have one schedule on 'backup_tool_perf', so I expect to see on team#2:
	// 	 - 'backup_tool_perf - 12 - $timestamp'
	stmt = `SELECT
		name,
		description,
		query,
		author_id,
		saved,
		observer_can_run,
		platform,
		min_osquery_version,
		schedule_interval,
		logging_type,
		automations_enabled
	FROM queries
	WHERE name LIKE ? AND team_id = ? AND name <> 'User 1 Query 2'`

	rows, err = db.Query(stmt, "per_query_perf%", 1)
	require.NoError(t, err)
	defer rows.Close()

	nRows = 0
	names = []string{}
	scheduleIntervals = []uint{}
	automationsEnabled = []bool{}
	loggingTypes = []string{}

	for rows.Next() {
		nRows += 1
		query := _20230719152138_Query{}
		err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.Platform,
			&query.MinOsqueryVersion,
			&query.ScheduleInterval,
			&query.LoggingType,
			&query.AutomationsEnabled,
		)
		require.NoError(t, err)

		names = append(names, query.Name)
		scheduleIntervals = append(scheduleIntervals, query.ScheduleInterval)
		automationsEnabled = append(automationsEnabled, query.AutomationsEnabled)
		loggingTypes = append(loggingTypes, query.LoggingType)

		require.Equal(t, query.Description, "Records the CPU time and memory usage for each individual query. Helpful for identifying queries that may impact performance.")
		require.Equal(t, query.Query, "SELECT name, interval, executions, output_size, wall_time, (user_time/executions) AS avg_user_time, (system_time/executions) AS avg_system_time, average_memory FROM osquery_schedule;")
		require.Equal(t, *query.AuthorID, uint(1))
		require.Equal(t, query.Saved, true)
		require.Equal(t, query.ObserverCanRun, false)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, names, []string{"per_query_perf - 11 - Jul 21 20:36:25.000"})
	require.ElementsMatch(t, scheduleIntervals, []uint{86400})
	require.ElementsMatch(t, automationsEnabled, []bool{true})
	require.ElementsMatch(t, loggingTypes, []string{"snapshot"})
	require.Equal(t, nRows, 1)
}
