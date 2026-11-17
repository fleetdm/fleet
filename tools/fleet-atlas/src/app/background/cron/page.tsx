import { CRONS, cadenceLabel } from "@/components/background/jobs";
import CronQuiz from "@/components/background/CronQuiz";
import Timeline from "@/components/background/Timeline";
import IntervalDefaultsMatchingQuiz from "@/components/interval-defaults-matching/IntervalDefaultsMatchingQuiz";
import { editorLink } from "@/lib/config";

export const instant = false;

const CRON_BY_CADENCE = () => {
  const sorted = [...CRONS].sort((a, b) => a.intervalMinutes - b.intervalMinutes);
  const buckets: Record<string, typeof CRONS> = {
    "Minute scale (< 10m)": [],
    "Minutes (10m to 60m)": [],
    "Hourly (1h to 24h)": [],
    "Daily+": [],
  };
  for (const c of sorted) {
    if (c.intervalMinutes < 10) buckets["Minute scale (< 10m)"].push(c);
    else if (c.intervalMinutes < 60) buckets["Minutes (10m to 60m)"].push(c);
    else if (c.intervalMinutes < 60 * 24) buckets["Hourly (1h to 24h)"].push(c);
    else buckets["Daily+"].push(c);
  }
  return buckets;
};

export default function CronPage() {
  const buckets = CRON_BY_CADENCE();
  return (
    <>
      <section className="mb-10">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
          24-hour cron timeline
        </h2>
        <p className="mb-3 text-xs text-zinc-500">
          Schematic. Dots are tick points for each cron over one day. Dense blue
          bars mean sub-minute ticks (so dots would overlap).
        </p>
        <Timeline />
      </section>

      <section>
        <h2 className="mb-4 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Cron catalog ({CRONS.length})
        </h2>
        {Object.entries(buckets).map(([label, items]) =>
          items.length === 0 ? null : (
            <div key={label} className="mb-6">
              <h3 className="mb-2 text-sm font-semibold text-zinc-600 dark:text-zinc-400">
                {label}
              </h3>
              <ul className="space-y-2">
                {items.map((c) => (
                  <li
                    key={c.name}
                    className="rounded border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950"
                  >
                    <div className="flex items-baseline justify-between gap-3">
                      <code className="font-mono text-sm text-zinc-900 dark:text-zinc-100">
                        {c.name}
                      </code>
                      <span className="text-xs text-zinc-500">
                        {cadenceLabel(c.intervalMinutes)}
                      </span>
                    </div>
                    <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                      {c.purpose}
                    </p>
                    <a
                      href={editorLink(c.file)}
                      className="mt-1 inline-block font-mono text-xs text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-300"
                    >
                      {c.file}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ),
        )}
      </section>

      <section id="cron-quiz" className="mt-10 scroll-mt-20">
        <CronQuiz />
      </section>

      <section id="interval-defaults-matching" className="mt-10 scroll-mt-20">
        <IntervalDefaultsMatchingQuiz />
      </section>
    </>
  );
}
