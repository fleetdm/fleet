import Link from "next/link";
import { TRACERS } from "@/components/tracer/tracers";
import AssessmentIcon from "@/components/AssessmentIcon";

export const instant = false;

const METHOD_COLOR: Record<string, string> = {
  GET: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300",
  POST: "bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300",
  PATCH: "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
  DELETE: "bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300",
};

export default function TracerIndex() {
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Request tracer
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Pick an endpoint; walk through every layer that touches it with real
          Fleet code. Builds the mental model of what happens when an HTTP
          request lands.
        </p>
      </header>

      <ul className="space-y-3">
        {TRACERS.map((t) => (
          <li key={t.slug}>
            <Link
              href={`/tracer/${t.slug}`}
              className={`block rounded border p-4 transition-colors ${
                t.stub
                  ? "border-dashed border-zinc-300 bg-white opacity-60 dark:border-zinc-800 dark:bg-zinc-950"
                  : "border-zinc-200 bg-white hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
              }`}
            >
              <div className="flex items-center gap-3">
                <span
                  className={`rounded px-2 py-0.5 font-mono text-xs ${
                    METHOD_COLOR[t.method] ?? ""
                  }`}
                >
                  {t.method}
                </span>
                <span className="font-mono text-sm text-zinc-700 dark:text-zinc-300">
                  {t.path}
                </span>
                {t.stub ? (
                  <span className="ml-auto rounded bg-zinc-200 px-2 py-0.5 text-xs text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
                    coming soon
                  </span>
                ) : (
                  <span className="ml-auto">
                    <AssessmentIcon />
                  </span>
                )}
              </div>
              <div className="mt-2 text-sm font-medium text-zinc-900 dark:text-zinc-100">
                {t.title}
              </div>
              <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                {t.summary}
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
