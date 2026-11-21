import Link from "next/link";

export const instant = false;

const SUBTABS = [
  {
    href: "/data/key-tables",
    label: "The 20 tables to know",
    blurb:
      "Curated short list for debugging. If you're going to learn only 20, learn these.",
    featured: true,
  },
  {
    href: "/data/mysql",
    label: "MySQL: Full catalog",
    blurb:
      "~240 tables grouped by topic. The source of truth: hosts, policies, software, MDM, activity.",
  },
  {
    href: "/data/redis",
    label: "Redis",
    blurb:
      "Distributed locks, live-query pubsub, rate limits, small shared caches.",
  },
  {
    href: "/data/filestore",
    label: "File store",
    blurb:
      "Blobs too big for MySQL: installers, bootstrap packages, profiles, scripts.",
  },
];

export default function DataLanding() {
  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Data layer
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Where Fleet keeps its state and why each store exists.
        </p>
      </header>
      <ul className="grid gap-4 md:grid-cols-2">
        {SUBTABS.map((s) => (
          <li key={s.href}>
            <Link
              href={s.href}
              className="block h-full rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
            >
              <div className="flex items-center gap-2">
                <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                  {s.label}
                </div>
                {s.featured ? (
                  <span className="rounded border border-emerald-300 bg-emerald-50 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-200">
                    Start here
                  </span>
                ) : null}
              </div>
              <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                {s.blurb}
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
