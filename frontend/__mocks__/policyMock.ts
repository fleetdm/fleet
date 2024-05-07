import { IPolicyStats } from "interfaces/policy";

const DEFAULT_POLICY_MOCK: IPolicyStats = {
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
  host_count_updated_at: "2023-11-30T19:05:13Z",
  webhook: "Off",
  has_run: true,
  next_update_ms: 3600000,
  calendar_events_enabled: true,
};

const createMockPolicy = (overrides?: Partial<IPolicyStats>): IPolicyStats => {
  return { ...DEFAULT_POLICY_MOCK, ...overrides };
};

export default createMockPolicy;
