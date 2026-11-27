// The six core osquery TLS endpoints Fleet implements. Matching quiz pairs:
// left = method + path (the thing you'd grep for), right = plain-language
// "what crosses this endpoint and why."

export type OsqueryEndpointPair = {
  key: string;
  method: "POST";
  path: string;
  description: string;
};

export const OSQUERY_ENDPOINTS: OsqueryEndpointPair[] = [
  {
    key: "enroll",
    method: "POST",
    path: "/api/osquery/enroll",
    description:
      "Device sends a shared enroll secret; Fleet returns a per-host node key (bearer token).",
  },
  {
    key: "config",
    method: "POST",
    path: "/api/osquery/config",
    description:
      "Device asks \"what's my config right now?\"; Fleet returns scheduled queries, options, labels.",
  },
  {
    key: "distributed-read",
    method: "POST",
    path: "/api/osquery/distributed/read",
    description:
      "Device asks \"is there anything I need to run right now?\"; Fleet returns zero or more live-query SQLs.",
  },
  {
    key: "distributed-write",
    method: "POST",
    path: "/api/osquery/distributed/write",
    description: "Device returns results from live-query runs.",
  },
  {
    key: "log",
    method: "POST",
    path: "/api/osquery/log",
    description:
      "Device returns results from scheduled queries, plus osquery's own logs.",
  },
  {
    key: "carve",
    method: "POST",
    path: "/api/osquery/carve/*",
    description:
      "File carving (osquery uploading a file in chunks).",
  },
];
