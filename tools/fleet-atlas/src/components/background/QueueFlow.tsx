import { editorLink } from "@/lib/config";
import type { QueueFlow, QueueNode } from "./queues";

const NODE_STYLES: Record<QueueNode["kind"], string> = {
  entry: "border-sky-400 bg-sky-50 dark:border-sky-700 dark:bg-sky-950/60",
  storage:
    "border-violet-400 bg-violet-50 dark:border-violet-700 dark:bg-violet-950/60",
  drain:
    "border-amber-400 bg-amber-50 dark:border-amber-700 dark:bg-amber-950/60",
  exit:
    "border-emerald-400 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/60",
};

const NODE_LABEL: Record<QueueNode["kind"], string> = {
  entry: "Entry",
  storage: "Queue storage",
  drain: "Drain",
  exit: "Exit",
};

const STORAGE_BADGE: Record<QueueFlow["storageType"], string> = {
  MySQL:
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  Redis:
    "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200",
  Memory: "bg-zinc-100 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200",
};

export default function QueueFlowCard({ flow }: { flow: QueueFlow }) {
  return (
    <section
      id={flow.slug}
      className="mb-10 scroll-mt-6 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
    >
      <header className="mb-4">
        <div className="flex items-baseline gap-3">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            {flow.label}
          </h2>
          <span
            className={`rounded px-2 py-0.5 text-[11px] font-semibold ${
              STORAGE_BADGE[flow.storageType]
            }`}
          >
            {flow.storageType}
          </span>
        </div>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          {flow.one_liner}
        </p>
      </header>

      <p className="mb-5 text-sm text-zinc-700 dark:text-zinc-300">
        {flow.overview}
      </p>

      <div className="flex flex-wrap items-stretch gap-2">
        {flow.nodes.map((n, i) => (
          <div key={i} className="flex items-stretch gap-2">
            <div
              className={`flex w-56 flex-col rounded border ${
                NODE_STYLES[n.kind]
              } p-3`}
            >
              <div className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-zinc-500">
                {NODE_LABEL[n.kind]}
              </div>
              <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                {n.label}
              </div>
              {n.detail ? (
                <div className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                  {n.detail}
                </div>
              ) : null}
              {n.file ? (
                <a
                  href={editorLink(n.file)}
                  className="mt-1 font-mono text-[11px] text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-300"
                >
                  {n.file}
                </a>
              ) : null}
            </div>
            {i < flow.nodes.length - 1 ? (
              <div className="flex items-center text-zinc-400 dark:text-zinc-600">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M4 11h12l-5-5 1.5-1.5L21 12l-8.5 7.5L11 18l5-5H4z" />
                </svg>
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}
