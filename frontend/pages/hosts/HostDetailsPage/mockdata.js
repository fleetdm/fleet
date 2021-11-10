const MOCK_DATA = {
  host: {
    id: 1,
    hostname: "noahs-macbook-pro.local",
    pack_stats: [
      {
        pack_id: 1,
        pack_name: "Global",
        type: "global",
        team_id: null,
        query_stats: [
          {
            query_name: "Get crashes",
            interval: 5148000,
            denylisted: false,
            executions: 1,
            system_time: 500,
            user_time: 500,
          },
          {
            query_name: "Detect machines with Gatekeeper disabled",
            interval: 600,
            denylisted: false,
            executions: 1,
            system_time: 1000,
            user_time: 2000,
          },
          {
            query_name: "Detect unencrypted SSH keys for local accounts",
            interval: 7200,
            denylisted: false,
            executions: 1,
            system_time: 3000,
            user_time: 2000,
          },
        ],
      },
      {
        pack_id: 2,
        pack_name: "Client Platform Engineering",
        type: "team-2",
        query_stats: [
          {
            query_name:
              "Detect dynamic linker hijacking on Linux (MITRE. T1574.006)",
            interval: 7200,
            denylisted: false,
            executions: 0,
            system_time: 0,
            user_time: 0,
          },
        ],
      },

      {
        pack_id: 5,
        pack_name: "Performance metrics",
        type: "pack",
        query_stats: [
          {
            query_name: "per_query_perf",
            interval: 5148000,
            denylisted: false,
            executions: 1,
            system_time: 500,
            user_time: 500,
          },
          {
            query_name: "runtime_perf",
            interval: 600,
            denylisted: false,
            executions: 1,
            system_time: 1000,
            user_time: 2000,
          },
          {
            query_name: "endpoint_security_tool_perf",
            interval: 7200,
            denylisted: false,
            executions: 1,
            system_time: 3000,
            user_time: 2000,
          },
        ],
      },
      {
        pack_id: 6,
        pack_name: "Security tooling checks",
        type: "pack",
        query_stats: [
          {
            query_name: "endpoint_security_tool_not_run",
            interval: 5148000,
            denylisted: false,
            executions: 0,
            system_time: 0,
            user_time: 0,
          },
          {
            query_name: "backup_tool_not_running",
            interval: 3600,
            denylisted: false,
            executions: 0,
            system_time: 0,
            user_time: 0,
          },
          {
            query_name: "endpoint_security_tool_perf",
            interval: 7200,
            denylisted: false,
            executions: 0,
            system_time: 0,
            user_time: 0,
          },
        ],
      },
    ],
  },
};

export default MOCK_DATA;
