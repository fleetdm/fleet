import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Live query flow. The full 9-step version. The two "why" moments most people
// skip:
//   - Fleet writes the campaign to MySQL first so every Fleet instance behind
//     the load balancer can see it (step 2).
//   - Hosts might POST results to a Fleet instance that ISN'T the one holding
//     the admin's WebSocket, so Redis pubsub bridges them (steps 7 and 8).

export const LIVE_QUERY_STEPS: SortStep[] = [
  {
    key: "admin-click-run",
    actor: "Admin",
    text: "Admin runs a live report from the UI, fleetctl, or the API.",
  },
  {
    key: "campaign-written-to-mysql",
    actor: "Fleet",
    text: "Fleet writes the campaign (distributed_query_campaigns row + targets) to MySQL so every Fleet instance behind the load balancer can see it.",
  },
  {
    key: "ws-subscribes",
    actor: "Fleet",
    text: "The admin's UI opens a WebSocket to Fleet; Fleet subscribes it to the Redis pubsub channel keyed by the campaign id.",
  },
  {
    key: "host-pulls-sql",
    actor: "Device",
    text: "On its next POST /api/osquery/distributed/read, each targeted host pulls the campaign SQL from whichever Fleet instance it hits.",
  },
  {
    key: "host-runs-sql",
    actor: "Device",
    text: "Host runs the SQL locally against its osquery tables.",
  },
  {
    key: "host-posts-rows",
    actor: "Device",
    text: "Host POSTs the result rows to whichever Fleet instance receives the request. Not necessarily the one holding the admin's WebSocket.",
  },
  {
    key: "fleet-publishes",
    actor: "Fleet",
    text: "That Fleet instance publishes the rows to the campaign's Redis pubsub channel.",
  },
  {
    key: "ws-server-receives",
    actor: "Fleet",
    text: "The Fleet instance holding the admin's WebSocket, subscribed to the same channel, receives the publish from Redis.",
  },
  {
    key: "ws-streams-browser",
    actor: "Fleet",
    text: "That WebSocket handler streams the rows to the admin's browser.",
  },
];
