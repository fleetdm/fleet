// Five Fleet webhook types matched to their characteristic profile.
// Left = webhook type name; right = "fires when / cadence / request shape."

export type WebhookTypePair = {
  key: string;
  name: string;
  profile: string;
};

export const WEBHOOK_TYPES: WebhookTypePair[] = [
  {
    key: "activity",
    name: "Activity",
    profile:
      "Any audited activity (admin logins, query edits, scripts run). Fires as activities occur. One request per activity.",
  },
  {
    key: "failing-policy",
    name: "Failing policy",
    profile:
      "A policy is newly failing on a host. Fleet checks daily by default. One request per newly failing host.",
  },
  {
    key: "vulnerability",
    name: "Vulnerability",
    profile:
      "A new CVE (published in the last 30 days by default) is detected. Fleet checks hourly by default. One request per host carrying the new CVE.",
  },
  {
    key: "host-status",
    name: "Host status",
    profile:
      "A set percentage of hosts has been offline for a set number of days. Fleet checks daily by default. One summary request covering the whole fleet.",
  },
  {
    key: "report-results",
    name: "Report results",
    profile:
      "A scheduled report produces results (webhook as log destination). Fires each report interval. One request per result batch.",
  },
];
