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
  team_id: null,
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
  install_software: {
    name: "testSw0",
    software_title_id: 1,
  },
};

const createMockPolicy = (overrides?: Partial<IPolicyStats>): IPolicyStats => {
  return { ...DEFAULT_POLICY_MOCK, ...overrides };
};

export const createMockPoliciesResponse = (
  overrides?: Partial<IPolicyStats>
) => {
  const MOCK_POLICIES_RESPONSE: { policies: IPolicyStats[] } = {
    policies: [
      {
        id: 5,
        name: "Gatekeeper enabled",
        query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
        description: "Checks if gatekeeper is enabled on macOS devices",
        critical: true,
        author_id: 42,
        author_name: "John",
        author_email: "john@example.com",
        team_id: 2,
        resolution: "Resolution steps",
        platform: "darwin",
        created_at: "2021-12-16T14:37:37Z",
        updated_at: "2021-12-16T16:39:00Z",
        passing_host_count: 2000,
        failing_host_count: 300,
        host_count_updated_at: "2023-12-20T15:23:57Z",
        webhook: "Off",
        has_run: true,
        next_update_ms: 3600000,
        calendar_events_enabled: false,
      },
      {
        id: 29090,
        name: "Windows machines with encrypted hard disks",
        query: "SELECT 1 FROM bitlocker_info WHERE protection_status = 1;",
        description: "Checks if the hard disk is encrypted on Windows devices",
        critical: false,
        author_id: 43,
        author_name: "Alice",
        author_email: "alice@example.com",
        team_id: 2,
        resolution: "Resolution steps",
        platform: "windows",
        created_at: "2021-12-16T14:37:37Z",
        updated_at: "2021-12-16T16:39:00Z",
        passing_host_count: 2300,
        failing_host_count: 0,
        host_count_updated_at: "2023-12-20T15:23:57Z",
        webhook: "Off",
        has_run: true,
        next_update_ms: 3600000,
        calendar_events_enabled: false,
      },
      {
        id: 136,
        name: "Arbitrary Test Policy (all platforms) (all teams)",
        query: "SELECT 1 FROM osquery_info WHERE 1=1;",
        description:
          "If you're seeing this, mostly likely this is because someone is testing out failing policies in dogfood. You can ignore this.",
        critical: true,
        author_id: 77,
        author_name: "Test Admin",
        author_email: "test@admin.com",
        team_id: null,
        resolution:
          'To make it pass, change "1=0" to "1=1". To make it fail, change "1=1" to "1=0".',
        platform: "darwin,windows,linux",
        created_at: "2022-08-04T19:30:18Z",
        updated_at: "2022-08-30T15:08:26Z",
        passing_host_count: 10,
        failing_host_count: 9,
        host_count_updated_at: "2023-12-20T15:23:57Z",
        webhook: "Off",
        has_run: true,
        next_update_ms: 3600000,
        calendar_events_enabled: false,
      },
    ],
  };

  if (overrides) {
    MOCK_POLICIES_RESPONSE.policies.push(createMockPolicy(overrides));
  }

  return MOCK_POLICIES_RESPONSE;
};

export default createMockPolicy;
