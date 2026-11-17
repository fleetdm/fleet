import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Logical dependency chain. Not a strict clock timeline. Each step reads
// what the previous step wrote:
//   - osquery can't check in until Orbit started it.
//   - Vitals + detail queries populate host_software before the vuln cron
//     can match it against CVE feeds.
//   - policy_membership has to exist before Fleet Desktop can surface a
//     failing policy to the user.
// Scheduled reports are intentionally not here. They run on their own
// cadence independent of this chain.

export const DAY_IN_THE_LIFE_STEPS: SortStep[] = [
  {
    key: "orbit-starts",
    actor: "Device",
    text: "Orbit starts osqueryd with the flags Fleet manages, and checks the TUF update server for signed component updates.",
  },
  {
    key: "osquery-enrolls",
    actor: "Device",
    text: "osquery enrolls with its enroll secret, gets back a node key, and begins checking in on its distributed_interval (10s default).",
  },
  {
    key: "detail-queries-populate-inventory",
    actor: "Device",
    text: "Fleet's built-in detail queries run on the next check-in; their results populate host_software + host OS / users / network tables.",
  },
  {
    key: "vuln-cron-reads-inventory",
    actor: "Fleet",
    text: "The vulnerabilities cron reads host_software, matches it against NVD / OSV / OVAL / MSRC feeds, and writes CVE matches to software_cve.",
  },
  {
    key: "policies-evaluate-and-fail",
    actor: "Device",
    text: "Policies run against osquery tables on the host; failing policies flip rows in policy_membership.",
  },
  {
    key: "fleet-desktop-surfaces-failing",
    actor: "External",
    text: "End user opens Fleet Desktop; it reads policy_membership and shows them which policies fail with resolution text.",
  },
];
