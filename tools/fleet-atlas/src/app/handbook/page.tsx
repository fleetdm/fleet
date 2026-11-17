import {
  CROSS_CUTTING,
  HANDBOOK_BY_SLUG,
} from "@/components/start/handbook-data";

export const instant = false;

export default function HandbookBridgePage() {
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Fleet handbook bridge
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Atlas tells you how Fleet (the product) works. The{" "}
          <a
            href="https://fleetdm.com/handbook"
            target="_blank"
            rel="noopener"
            className="underline hover:text-zinc-900 dark:hover:text-zinc-200"
          >
            Fleet handbook
          </a>{" "}
          tells you how Fleet (the company) works. Rituals, process, writing style,
          security posture, release cadence. Both are public. Read both. This page
          is the per-role shortlist, so you don&apos;t have to guess which handbook
          section to start with.
        </p>
      </header>

      <section className="mb-10">
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Everyone should read these
        </h2>
        <ul className="grid gap-3 md:grid-cols-2">
          {CROSS_CUTTING.map((h) => (
            <li key={h.href}>
              <a
                href={h.href}
                target="_blank"
                rel="noopener"
                className="block h-full rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
              >
                <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                  {h.title} ↗
                </div>
                <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                  {h.blurb}
                </p>
              </a>
            </li>
          ))}
        </ul>
      </section>

      {Object.values(HANDBOOK_BY_SLUG).map((r) => (
        <section key={r.roleLabel} className="mb-10">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
            {r.roleLabel}
          </h2>
          <p className="mb-4 mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            {r.intro}
          </p>
          <ul className="space-y-2">
            {r.pointers.map((h) => (
              <li key={h.href}>
                <a
                  href={h.href}
                  target="_blank"
                  rel="noopener"
                  className="block rounded border border-zinc-200 bg-white p-3 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                >
                  <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                    {h.title} ↗
                  </div>
                  <p className="mt-0.5 text-xs text-zinc-600 dark:text-zinc-400">
                    {h.blurb}
                  </p>
                </a>
              </li>
            ))}
          </ul>
        </section>
      ))}

      <div className="rounded border border-zinc-200 bg-zinc-50 p-4 text-xs text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
        Something missing? The Fleet handbook changes constantly. Open{" "}
        <a
          href="https://fleetdm.com/handbook"
          target="_blank"
          rel="noopener"
          className="underline"
        >
          fleetdm.com/handbook
        </a>{" "}
        for the full index.
      </div>
    </div>
  );
}
