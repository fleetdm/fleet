"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import {
  SCENARIOS,
  type Scenario,
  type ScenarioTag,
} from "@/components/scenarios/scenarios";
import TagChip from "@/components/scenarios/TagChip";
import AssessmentPanel from "@/components/start/AssessmentPanel";

type Props = { customScenarios: Scenario[] };

export default function ScenariosClient({ customScenarios }: Props) {
  const [active, setActive] = useState<ScenarioTag | null>(null);
  const [query, setQuery] = useState("");

  const combined = useMemo(
    () => [
      ...customScenarios.map((s) => ({ ...s, custom: true })),
      ...SCENARIOS.map((s) => ({ ...s, custom: false })),
    ],
    [customScenarios],
  );

  const allTags = useMemo(() => {
    const set = new Set<ScenarioTag>();
    for (const s of combined) for (const t of s.tags) set.add(t);
    return Array.from(set).sort();
  }, [combined]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return combined.filter((s) => {
      if (active && !s.tags.includes(active)) return false;
      if (q && !s.label.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [active, combined, query]);

  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Scenarios
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Guided simulations. Pick a flow; watch the step-by-step from
          external actor to Fleet to data store and back. Builds the mental
          model that an architecture map alone can&apos;t.
        </p>
      </header>

      <div className="mb-4">
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search scenarios by title..."
          className="w-full rounded border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-zinc-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus:border-zinc-400"
        />
      </div>

      <div className="mb-6 flex flex-wrap items-center gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Filter
        </span>
        {allTags.map((t) => (
          <TagChip
            key={t}
            tag={t}
            as="button"
            active={active === t}
            onClick={() => setActive(active === t ? null : t)}
          />
        ))}
        {active ? (
          <button
            onClick={() => setActive(null)}
            className="cursor-pointer rounded border border-zinc-300 px-2 py-0.5 text-[11px] text-zinc-600 hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-400 dark:hover:bg-zinc-800"
          >
            clear
          </button>
        ) : null}
      </div>

      <ul className="space-y-4">
        {visible.map((s) => (
          <li key={s.slug}>
            <Link
              href={`/scenarios/${s.slug}`}
              className="block rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
            >
              <div className="flex items-center gap-2">
                <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                  {s.label}
                </div>
                {s.custom ? (
                  <span className="rounded border border-zinc-300 px-1.5 text-[10px] font-medium uppercase tracking-wide text-zinc-500 dark:border-zinc-700">
                    local
                  </span>
                ) : null}
              </div>
              <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                {s.summary}
              </p>
              {s.tags.length > 0 ? (
                <div className="mt-3 flex flex-wrap gap-1">
                  {s.tags.map((t) => (
                    <TagChip key={t} tag={t} />
                  ))}
                </div>
              ) : null}
            </Link>
          </li>
        ))}
      </ul>

      {visible.length === 0 ? (
        <p className="mt-6 text-sm text-zinc-500">
          No scenarios match {query && active ? "that search and tag" : query ? "that search" : "that tag"} yet.
        </p>
      ) : null}

      <AssessmentPanel
        assessments={[
          { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
          { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
          { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
          { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
        ]}
      />
    </div>
  );
}
