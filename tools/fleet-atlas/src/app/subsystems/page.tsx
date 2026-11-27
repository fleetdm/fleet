"use client";

import Link from "next/link";
import { useState } from "react";
import {
  SUBSYSTEMS,
  SUBSYSTEM_CATEGORIES,
  type SubsystemCategory,
} from "@/components/subsystems/subsystems";
import AssessmentIcon from "@/components/AssessmentIcon";
import { subsystemHasEmbeddedQuiz } from "@/components/subsystems/embedded-quizzes";

export default function SubsystemsIndex() {
  const [filter, setFilter] = useState<SubsystemCategory | "all">("all");

  const visible = SUBSYSTEMS.filter(
    (s) => filter === "all" || s.category === filter,
  );

  // Group visible subsystems by category, preserving the order declared in
  // SUBSYSTEM_CATEGORIES. Featured items (overviews) float to the top of their
  // category.
  const grouped = SUBSYSTEM_CATEGORIES.map((cat) => ({
    category: cat,
    items: visible
      .filter((s) => s.category === cat)
      .sort((a, b) => Number(!!b.featured) - Number(!!a.featured)),
  })).filter((g) => g.items.length > 0);

  const counts = SUBSYSTEM_CATEGORIES.map((cat) => ({
    cat,
    n: SUBSYSTEMS.filter((s) => s.category === cat).length,
  }));

  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Subsystems
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet is a lot of smaller systems bolted together. Each page here
          takes one and tells its lifecycle, key files, and the edge cases that
          bite.
        </p>
      </header>

      <div className="mb-6 flex flex-wrap gap-2 text-xs">
        <button
          onClick={() => setFilter("all")}
          className={`rounded border px-2 py-0.5 ${
            filter === "all"
              ? "border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
              : "border-zinc-300 bg-white text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          }`}
        >
          All ({SUBSYSTEMS.length})
        </button>
        {counts.map(({ cat, n }) => (
          <button
            key={cat}
            onClick={() => setFilter(cat === filter ? "all" : cat)}
            className={`rounded border px-2 py-0.5 ${
              filter === cat
                ? "border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
                : "border-zinc-300 bg-white text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
            }`}
          >
            {cat} ({n})
          </button>
        ))}
      </div>

      {grouped.map((g) => (
        <section key={g.category} className="mb-8">
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
            {g.category}
          </h2>
          <ul className="grid gap-4 md:grid-cols-2">
            {g.items.map((s) => (
              <li key={s.slug}>
                <Link
                  href={`/subsystems/${s.slug}`}
                  className="block h-full rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                >
                  <div className="flex items-center gap-2">
                    <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                      {s.label}
                    </div>
                    {s.featured && (
                      <span className="rounded border border-sky-300 bg-sky-50 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-sky-800 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200">
                        Start here
                      </span>
                    )}
                    {subsystemHasEmbeddedQuiz(s.slug) && (
                      <span className="ml-auto">
                        <AssessmentIcon />
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                    {s.one_liner}
                  </p>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
