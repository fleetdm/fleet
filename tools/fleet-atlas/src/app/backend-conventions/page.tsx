"use client";

import { useMemo, useState } from "react";
import { ISMS, type FleetIsm } from "@/components/fleetisms/isms";

type Category = FleetIsm["category"];

const CATEGORY_COLORS: Record<Category, string> = {
  Terminology:
    "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  "Error handling":
    "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200",
  Authorization:
    "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  Datastore:
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  "Service layer":
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  Testing:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  Migrations:
    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/60 dark:text-cyan-200",
  "Review hygiene":
    "bg-zinc-200 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200",
};

export default function FleetIsmsPage() {
  const [active, setActive] = useState<Category | null>(null);

  const categories = useMemo(() => {
    const set = new Set<Category>();
    for (const i of ISMS) set.add(i.category);
    return Array.from(set);
  }, []);

  const visible = active ? ISMS.filter((i) => i.category === active) : ISMS;

  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <header className="mb-6 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Backend conventions
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          House conventions. Read these before you open your first backend PR.
          {ISMS.length} rules.
        </p>
      </header>

      <div className="mb-6 flex flex-wrap items-center gap-2 text-xs">
        <button
          onClick={() => setActive(null)}
          className={`rounded border px-2 py-0.5 ${
            active === null
              ? "border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
              : "border-zinc-300 bg-white text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          }`}
        >
          All
        </button>
        {categories.map((c) => (
          <button
            key={c}
            onClick={() => setActive(active === c ? null : c)}
            className={`inline-flex items-center gap-1 rounded px-2 py-0.5 ${CATEGORY_COLORS[c]} ${
              active === c ? "ring-1 ring-zinc-900 dark:ring-zinc-100" : ""
            }`}
          >
            {c}
          </button>
        ))}
      </div>

      <ol className="space-y-5">
        {visible.map((i, idx) => (
          <li
            key={i.slug}
            id={i.slug}
            className="scroll-mt-6 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
          >
            <div className="flex items-baseline gap-3">
              <span className="font-mono text-xs text-zinc-500">
                {String(idx + 1).padStart(2, "0")}
              </span>
              <h2 className="flex-1 text-base font-semibold text-zinc-900 dark:text-zinc-100">
                {i.title}
              </h2>
              <span
                className={`shrink-0 rounded px-2 py-0.5 text-[10px] font-semibold ${CATEGORY_COLORS[i.category]}`}
              >
                {i.category}
              </span>
            </div>

            <div className="mt-3 border-l-2 border-zinc-300 pl-3 text-sm text-zinc-800 dark:border-zinc-700 dark:text-zinc-200">
              {i.rule}
            </div>

            <p className="mt-3 text-xs text-zinc-600 dark:text-zinc-400">
              <span className="font-semibold uppercase tracking-wide">
                Why:
              </span>{" "}
              {i.why}
            </p>

            {i.example ? (
              <pre className="mt-3 overflow-x-auto rounded bg-zinc-100 p-3 text-xs text-zinc-900 dark:bg-zinc-900 dark:text-zinc-100">
                {i.example}
              </pre>
            ) : null}

            {i.gotchas && i.gotchas.length > 0 ? (
              <div className="mt-3">
                <div className="text-xs font-semibold uppercase tracking-wide text-rose-700 dark:text-rose-400">
                  Common failure modes
                </div>
                <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-zinc-700 dark:text-zinc-300">
                  {i.gotchas.map((g, idx2) => (
                    <li key={idx2}>{g}</li>
                  ))}
                </ul>
              </div>
            ) : null}
          </li>
        ))}
      </ol>
    </div>
  );
}
