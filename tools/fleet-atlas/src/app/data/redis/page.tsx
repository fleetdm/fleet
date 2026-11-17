import Link from "next/link";
import Markdown from "@/components/Markdown";

export const instant = false;

const USES = [
  {
    slug: "locks",
    label: "Distributed locks (crons)",
    tooltip:
      "Named Redis keys set with SETNX + TTL so only one Fleet instance runs each cron per tick.",
    body: `
Fleet runs the same cron code on every server instance. Only one should actually do the work per tick. Fleet implements this with a Redis **SETNX with expiry** pattern: the first instance to set the key wins; the key auto-expires so a crashed instance can&apos;t deadlock the job.

**Key shape:** \`<cron-name>:lock\` or \`calendar:lock:<event-uuid>\` (per-event locking)

**Code pointers**
- Generic cron lock: \`server/service/schedule/schedule.go\`
- Example per-item lock: \`server/service/calendar/calendar.go\` constants \`LockKeyPrefix\` and \`ReservedLockKeyPrefix\`.
`,
  },
  {
    slug: "pubsub",
    label: "Live query pubsub",
    tooltip:
      "Fan-out of live-query results across Fleet instances: each campaign has one Redis channel; result-receiving instances publish, WebSocket-holding instances subscribe.",
    body: `
Live queries fan out to many hosts; results trickle in over seconds to minutes. Rather than have the admin&apos;s UI poll MySQL, Fleet publishes each result to a Redis pubsub channel keyed by the campaign UUID, and the UI&apos;s WebSocket subscribes.

**Why this matters**: in a multi-instance Fleet, instance A may receive a host&apos;s result over osquery TLS, while instance B holds the admin&apos;s WebSocket. Redis pubsub bridges them.

**Channel shape:** one channel per \`distributed_query_campaign\` ID.

**Code pointers**
- \`server/pubsub/redis_query_results.go\`
- \`server/pubsub/inmem_query_results.go\` (in-memory fallback for single-instance dev)
`,
  },
  {
    slug: "rate-limits",
    label: "Rate limits",
    tooltip:
      "Sliding-window counters (INCR + EXPIRE) for login attempts, forgot-password, SSO callback, and other auth-adjacent endpoints.",
    body: `
Fleet rate-limits certain endpoints (login attempts, forgot-password, SSO callback) with sliding-window counters in Redis. Simple INCR with EXPIRE pattern.

**Why Redis**: counters mutate thousands of times a second. Doing it in MySQL would hammer the write path.
`,
  },
  {
    slug: "caches",
    label: "Short-lived caches",
    tooltip:
      "Grab-bag of small-value keys with short TTLs: nonce stores (SCIM, ACME, PSSO), de-dup keys, cached config computations.",
    body: `
A grab-bag of small keys with short TTLs: cached config computations, nonce stores (SCIM, ACME, PSSO), de-duplication keys for notifications.

**Code pointers**
- \`server/mdm/acme/internal/redis_nonces_store/\`
- \`server/mdm/psso/internal/redis_nonces_store/\`
`,
  },
];

export default function RedisPage() {
  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link href="/data" className="hover:text-zinc-900 dark:hover:text-zinc-200">
          ← Data layer
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Redis
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet uses Redis for coordination and messaging, not primarily as a
          cache. It&apos;s a hard dependency: Fleet won&apos;t run without it.
        </p>
      </header>

      {USES.map((u) => (
        <section
          key={u.slug}
          className="mb-8 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
        >
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            {u.label}
          </h2>
          <div className="mt-3">
            <Markdown>{u.body}</Markdown>
          </div>
        </section>
      ))}
    </div>
  );
}
