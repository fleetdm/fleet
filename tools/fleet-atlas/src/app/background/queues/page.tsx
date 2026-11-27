import Link from "next/link";
import QueueFlowCard from "@/components/background/QueueFlow";
import { QUEUES } from "@/components/background/queues";
import QueueStorageMatchingQuiz from "@/components/queue-storage-matching/QueueStorageMatchingQuiz";
import AssessmentIcon from "@/components/AssessmentIcon";

export const instant = false;

// Each flow quiz that drills into one specific queue's end-to-end lifecycle.
// Linked out from under the matching quiz so a reader going deep on one queue
// can jump to its full sort on the relevant subsystem page.
const RELATED_FLOW_QUIZZES: Array<{
  label: string;
  href: string;
  blurb: string;
}> = [
  {
    label: "VPP install: Flow order",
    href: "/subsystems/software-install-vpp-mac#vpp-install-sort",
    blurb:
      "Seven steps from admin click through the apple_software worker job to Apple's App Store delivery.",
  },
  {
    label: "Custom installer: Lifecycle order",
    href: "/subsystems/software-install-custom#custom-install-sort",
    blurb:
      "Nine steps. Pull-based through fleetd's osquery /config poll; no MDM in the picture.",
  },
  {
    label: "Live query: Flow order",
    href: "/subsystems/live-queries#live-query-sort",
    blurb:
      "Nine steps including the MySQL campaign-write and the Redis pubsub + WebSocket bridge between Fleet instances.",
  },
  {
    label: "Mac DEP enrollment: Flow order",
    href: "/subsystems/apple-mdm#mac-dep-sort",
    blurb:
      "Uses the Apple MDM command queue end-to-end (nano_commands → APNs push → device MDM poll).",
  },
  {
    label: "Setup experience install: Flow order",
    href: "/subsystems/apple-mdm#setup-experience-sort",
    blurb:
      "Shows the Apple MDM command queue delivering a bootstrap package during the setup-assistant pause.",
  },
];

export default function QueuesPage() {
  return (
    <>
      <section className="mb-8 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
        <p className="text-sm text-zinc-700 dark:text-zinc-300">
          Fleet has more than one queue. Each has a different storage shape,
          drain mechanism, and failure mode. The pattern is always the same:
          something writes, something waits, something else drains.
        </p>
      </section>

      <nav className="mb-8 flex flex-wrap gap-2 text-xs">
        {QUEUES.map((q) => (
          <a
            key={q.slug}
            href={`#${q.slug}`}
            className="rounded border border-zinc-300 bg-white px-2 py-1 text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          >
            {q.label}
          </a>
        ))}
      </nav>

      {QUEUES.map((q) => (
        <QueueFlowCard key={q.slug} flow={q} />
      ))}

      <section
        id="queue-storage-matching"
        className="mt-10 scroll-mt-20"
      >
        <QueueStorageMatchingQuiz />
      </section>

      <section className="mt-10">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Go deeper. Related flow quizzes
        </h2>
        <p className="mb-3 text-xs text-zinc-600 dark:text-zinc-400">
          Several of these queues are the plumbing behind end-to-end user flows
          that have their own full sort quizzes on other pages. Walk the queue
          here, then prove you know its lifecycle there.
        </p>
        <ul className="space-y-2">
          {RELATED_FLOW_QUIZZES.map((q) => (
            <li key={q.href}>
              <Link
                href={q.href}
                className="block rounded border border-zinc-200 bg-white p-3 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
              >
                <div className="flex items-center gap-2">
                  <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                    {q.label}
                  </div>
                  <span className="ml-auto">
                    <AssessmentIcon />
                  </span>
                </div>
                <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                  {q.blurb}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
