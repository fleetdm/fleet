import { IConfig } from "interfaces/config";

const DEFAULT_CONFIG_MOCK: IConfig = {
  id: 1,
  name: "Antivirus healthy (Linux)",
  query:
    "SELECT score FROM (SELECT case when COUNT(*) = 2 then 1 ELSE 0 END AS score FROM processes WHERE (name = 'clamd') OR (name = 'freshclam')) WHERE score == 1;",
  critical: false,
  description:
    "Checks that both ClamAV's daemon and its updater service (freshclam) are running.",
  author_id: 1,
  author_name: "Test User",
  author_email: "test@user.com",
  team_id: undefined,
  resolution: "Ensure ClamAV and Freshclam are installed and running.",
  platform: "linux" as const,
  created_at: "2023-03-24T22:13:59Z",
  updated_at: "2023-03-31T19:05:13Z",
  passing_host_count: 0,
  failing_host_count: 8,
  webhook: "Off",
  has_run: true,
  osquery_policy_ms: 3600000,
};

const createMockConfig = (overrides?: Partial<IConfig>): IConfig => {
  return { ...DEFAULT_CONFIG_MOCK, ...overrides };
};

export default createMockConfig;
