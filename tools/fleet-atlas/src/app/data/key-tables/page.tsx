import Link from "next/link";
import {
  DOMAIN_ORDER,
  KEY_TABLES,
} from "@/components/data/key-tables";

export const instant = false;

const DOMAIN_COLORS: Record<string, string> = {
  Hosts:
    "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200",
  MDM:
    "border-rose-300 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950/60 dark:text-rose-200",
  Software:
    "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-200",
  Policies:
    "border-indigo-300 bg-indigo-50 text-indigo-800 dark:border-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-200",
  "Queue plumbing":
    "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950/60 dark:text-amber-200",
  "Audit & identity":
    "border-violet-300 bg-violet-50 text-violet-800 dark:border-violet-700 dark:bg-violet-950/60 dark:text-violet-200",
};

export default function KeyTablesPage() {
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link href="/data" className="hover:text-zinc-900 dark:hover:text-zinc-200">
          ← Data layer
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          The {KEY_TABLES.length} tables to know
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet&rsquo;s MySQL has ~240 tables. Of those, this is the short list
          that comes up in almost every &ldquo;where does X live?&rdquo; debugging
          conversation. Learn these, and you can trace most support tickets
          back to the right row.
        </p>
      </header>

      {DOMAIN_ORDER.map((domain) => {
        const tables = KEY_TABLES.filter((t) => t.domain === domain);
        if (tables.length === 0) return null;
        return (
          <section key={domain} className="mb-10">
            <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-zinc-500">
              {domain}
            </h2>
            <ul className="space-y-5">
              {tables.map((t) => (
                <li
                  key={t.name}
                  className="rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
                >
                  <div className="mb-2 flex items-center gap-3">
                    <code className="rounded bg-zinc-100 px-2 py-0.5 font-mono text-sm text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100">
                      {t.name}
                    </code>
                    <span
                      className={`rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
                        DOMAIN_COLORS[t.domain] ?? ""
                      }`}
                    >
                      {t.domain}
                    </span>
                  </div>
                  <p className="text-sm text-zinc-800 dark:text-zinc-200">
                    {t.oneLine}
                  </p>
                  <div className="mt-3">
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                      You&apos;d query this when...
                    </div>
                    <ul className="list-disc pl-5 text-sm text-zinc-700 dark:text-zinc-300">
                      {t.whenYouQuery.map((w, i) => (
                        <li key={i}>{w}</li>
                      ))}
                    </ul>
                  </div>
                  {t.notableColumns && t.notableColumns.length > 0 ? (
                    <div className="mt-3">
                      <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                        Notable columns
                      </div>
                      <ul className="space-y-0.5 text-xs font-mono text-zinc-700 dark:text-zinc-300">
                        {t.notableColumns.map((c, i) => (
                          <li key={i}>{c}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </li>
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
