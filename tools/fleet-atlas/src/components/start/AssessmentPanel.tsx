"use client";

import Link from "next/link";
import { useState } from "react";
import type { AssessmentRef } from "./paths";

type StoredEntry = {
  bucketLabel?: string;
  correct?: number;
  total?: number;
  finalDifficulty?: number;
  bestCorrect?: number;
  bestTotal?: number;
  bestFinalDifficulty?: number;
  bestAttempts?: number;
};

const STORAGE_KEYS: Record<AssessmentRef["surface"], string> = {
  glossary: "fleet-atlas:quiz-results:glossary",
  architecture: "fleet-atlas:quiz-results:architecture",
  tracer: "fleet-atlas:tracer-quiz-best",
};

function hrefFor(a: AssessmentRef): string {
  if (a.surface === "tracer") return `/tracer/${a.bucketSlug}`;
  const base =
    a.surface === "glossary"
      ? "/glossary/self-assessments"
      : "/architecture/self-assessments";
  return `${base}?bucket=${encodeURIComponent(a.bucketSlug)}`;
}

function readSurface(
  surface: AssessmentRef["surface"],
): Record<string, StoredEntry> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(STORAGE_KEYS[surface]);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, StoredEntry>;
    if (surface !== "tracer") return parsed;
    // Tracer records store { bestCorrect, total, bestAttempts, bestTakenAt }.
    // Normalize so the panel's bestCorrect / bestTotal rendering works.
    const out: Record<string, StoredEntry> = {};
    for (const [k, v] of Object.entries(parsed)) {
      out[k] = {
        ...v,
        bestTotal: v.total,
      };
    }
    return out;
  } catch {
    return {};
  }
}

function pct(c: number | undefined, t: number | undefined) {
  if (!c || !t) return 0;
  return Math.round((c / t) * 100);
}

export default function AssessmentPanel({
  assessments,
}: {
  assessments: AssessmentRef[];
}) {
  const ssr = typeof window === "undefined";
  const [glossary] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("glossary"),
  );
  const [architecture] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("architecture"),
  );
  const [tracer] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("tracer"),
  );

  if (assessments.length === 0) return null;

  return (
    <section className="mt-6 rounded border border-emerald-200 bg-emerald-50/40 p-4 dark:border-emerald-800 dark:bg-emerald-950/20">
      <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
        Prove you know it
      </h3>
      <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
        Assessments relevant to this path. Scores persist across visits; your
        best run per quiz is saved separately from your most recent.
      </p>
      <ul className="mt-3 space-y-2">
        {assessments.map((a) => {
          const store =
            a.surface === "glossary"
              ? glossary
              : a.surface === "architecture"
                ? architecture
                : tracer;
          const r = store[a.bucketSlug];
          const lastPct = pct(r?.correct, r?.total);
          const bestPct = pct(r?.bestCorrect, r?.bestTotal);
          return (
            <li key={`${a.surface}:${a.bucketSlug}`}>
              <Link
                href={hrefFor(a)}
                className="group block rounded border border-zinc-200 bg-white p-3 text-sm transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
              >
                <div className="flex items-center justify-between gap-3">
                  <span className="font-medium text-zinc-900 dark:text-zinc-100">
                    {a.bucketLabel}
                  </span>
                  <div className="flex items-center gap-2 text-xs">
                    {r?.bestCorrect ? (
                      <span className="rounded bg-emerald-100 px-2 py-0.5 font-semibold text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200">
                        Best {r.bestCorrect}/{r.bestTotal} ({bestPct}%)
                      </span>
                    ) : (
                      <span className="rounded border border-dashed border-zinc-300 px-2 py-0.5 text-zinc-500 dark:border-zinc-700">
                        Not taken yet
                      </span>
                    )}
                    {r?.correct && r.correct !== r.bestCorrect ? (
                      <span className="rounded bg-zinc-100 px-2 py-0.5 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                        Last {r.correct}/{r.total} ({lastPct}%)
                      </span>
                    ) : null}
                  </div>
                </div>
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
