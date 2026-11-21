// Fleet's most-asked-about default intervals, caps, and cadences. Left = the
// config knob or behavior name; right = its default value with a one-line
// rationale. The numbers are easy to look up but hard to retain unless you've
// seen them in context.

export type IntervalPair = {
  key: string;
  knob: string;
  value: string;
};

export const INTERVAL_DEFAULTS: IntervalPair[] = [
  {
    key: "distributed-interval",
    knob: "distributed_interval",
    value:
      "10 seconds. How often osquery checks POST /api/osquery/distributed/read for pending live-query work.",
  },
  {
    key: "detail-update-interval",
    knob: "osquery_detail_update_interval",
    value:
      "1 hour. Fleet re-asks hosts for vitals + software inventory when their data is older than this.",
  },
  {
    key: "policy-update-interval",
    knob: "osquery_policy_update_interval",
    value:
      "1 hour. Minimum time between runs of each policy's SQL per host.",
  },
  {
    key: "result-cap",
    knob: "Scheduled report stored-result cap",
    value:
      "1,000 results per report. Fleet drops oldest rows beyond this unless the destination handles history.",
  },
  {
    key: "splay",
    knob: "Scheduled report interval splay",
    value:
      "Up to 10% random per-host delay. Spreads load so hosts don't all report at the same instant.",
  },
  {
    key: "policy-webhook-check",
    knob: "Policy failing-webhook check interval",
    value:
      "24 hours. Fleet looks for newly failing policies once per day before firing their webhooks or tickets.",
  },
];
