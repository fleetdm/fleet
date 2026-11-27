"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import {
  allBuckets,
  STORAGE_KEYS,
  type SurfaceKind,
  type UnifiedBucket,
} from "@/lib/all-buckets";
import QuizFormatChip from "@/components/QuizFormatChip";
import {
  QUIZ_FORMAT_COLOR,
  QUIZ_FORMAT_LABEL,
  SURFACE_TO_FORMAT,
  type QuizFormat,
} from "@/components/quiz-formats";

type StoredEntry = {
  bucketLabel?: string;
  correct?: number;
  total?: number;
  finalDifficulty?: number;
  bestCorrect?: number;
  bestTotal?: number;
  bestFinalDifficulty?: number;
  takenAt?: number;
  bestTakenAt?: number;
  bestAttempts?: number;
};

function readSurface(s: SurfaceKind): Record<string, StoredEntry> {
  if (typeof window === "undefined") return {};
  try {
    // Matching has multiple buckets, each with its own storage key, so
    // handle it before the generic early-return on a missing key.
    if (s === "matching") {
      const results: Record<string, StoredEntry> = {};
      const matchingPairs: Array<[string, string]> = [
        ["glossary-all", STORAGE_KEYS[s]],
        ["osquery-tls", "fleet-atlas:osquery-tls-matching-quiz-best"],
        ["webhook-types", "fleet-atlas:webhook-types-matching-quiz-best"],
        [
          "interval-defaults",
          "fleet-atlas:interval-defaults-matching-quiz-best",
        ],
        ["queue-storage", "fleet-atlas:queue-storage-matching-quiz-best"],
      ];
      for (const [slug, key] of matchingPairs) {
        const raw = localStorage.getItem(key);
        if (raw) results[slug] = JSON.parse(raw) as StoredEntry;
      }
      return results;
    }
    // Flow surface has multiple sort buckets, one per flow, each with its
    // own storage key. Read each one independently.
    if (s === "flow") {
      const results: Record<string, StoredEntry> = {};
      const flowPairs: Array<[string, string]> = [
        ["sso-saml", "fleet-atlas:sso-saml-sort-quiz-best"],
        ["mac-dep", "fleet-atlas:mac-dep-sort-quiz-best"],
        ["setup-experience", "fleet-atlas:setup-experience-sort-quiz-best"],
        ["live-query", "fleet-atlas:live-query-sort-quiz-best"],
        ["vpp-install", "fleet-atlas:vpp-install-sort-quiz-best"],
        ["managed-google-play", "fleet-atlas:managed-google-play-sort-quiz-best"],
        ["inhouse-install", "fleet-atlas:inhouse-install-sort-quiz-best"],
        ["custom-install", "fleet-atlas:custom-install-sort-quiz-best"],
        ["day-in-the-life", "fleet-atlas:day-in-the-life-sort-quiz-best"],
      ];
      for (const [slug, key] of flowPairs) {
        const raw = localStorage.getItem(key);
        if (raw) results[slug] = JSON.parse(raw) as StoredEntry;
      }
      return results;
    }
    const raw = localStorage.getItem(STORAGE_KEYS[s]);
    if (!raw) return {};
    if (s === "tracer") {
      const parsed = JSON.parse(raw) as Record<string, StoredEntry>;
      const out: Record<string, StoredEntry> = {};
      for (const [k, v] of Object.entries(parsed)) {
        out[k] = { ...v, bestTotal: v.total };
      }
      return out;
    }
    if (s === "tier") {
      // Tier quiz stores a single flat record, not keyed by slug.
      const record = JSON.parse(raw) as StoredEntry;
      return {
        "free-vs-premium": { ...record, bestTotal: record.total },
      };
    }
    if (s === "review-ai") {
      const record = JSON.parse(raw) as StoredEntry;
      return { all: record };
    }
    if (s === "values") {
      const record = JSON.parse(raw) as StoredEntry;
      return { all: record };
    }
    if (s === "cron") {
      const record = JSON.parse(raw) as StoredEntry;
      return { "which-cron": record };
    }
    return JSON.parse(raw) as Record<string, StoredEntry>;
  } catch {
    return {};
  }
}

function pct(c: number | undefined, t: number | undefined) {
  if (!c || !t) return 0;
  return Math.round((c / t) * 100);
}

function statusOf(
  e: StoredEntry | undefined,
  surface: SurfaceKind,
  bucketQuestionCount: number,
): {
  label: string;
  color: string;
  criteria: string;
} {
  if (!e || !e.bestCorrect || !e.bestTotal)
    return {
      label: "Not taken",
      color:
        "border-zinc-300 bg-white text-zinc-500 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400",
      criteria: "Not taken: no completed attempt recorded yet.",
    };
  if (surface === "tracer") {
    if (e.bestCorrect === e.bestTotal)
      return {
        label: e.bestAttempts === 1 ? "Mastered" : "Solved",
        color:
          e.bestAttempts === 1
            ? "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200"
            : "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria:
          e.bestAttempts === 1
            ? "Mastered: ordered every step correctly on the first attempt."
            : "Solved: ordered every step correctly, took more than one attempt.",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria:
        "In progress: at least one attempt, but no run has placed every step correctly yet.",
    };
  }
  if (surface === "matching") {
    const p = pct(e.bestCorrect, e.bestTotal);
    if (p >= 90)
      return {
        label: "Mastered",
        color:
          "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
        criteria:
          "Mastered: best cumulative run ≥ 90% pairs matched correctly.",
      };
    if (p >= 70)
      return {
        label: "Solid",
        color:
          "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria:
          "Solid: best cumulative run ≥ 70% pairs matched correctly.",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria: "In progress: best cumulative run below 70% pairs correct.",
    };
  }
  if (surface === "flow") {
    if (e.bestCorrect === e.bestTotal && (e.bestAttempts ?? 2) === 1)
      return {
        label: "Mastered",
        color:
          "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
        criteria: "Mastered: all steps in order on the first attempt.",
      };
    if (e.bestCorrect === e.bestTotal)
      return {
        label: "Solved",
        color:
          "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria:
          "Solved: all steps in order, took more than one attempt.",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria:
        "In progress: at least one attempt; no run has placed every step in the right position yet.",
    };
  }
  if (surface === "values") {
    if (e.bestCorrect === e.bestTotal)
      return {
        label: "Mastered",
        color:
          "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
        criteria: "Mastered: named all 5 Fleet values correctly in one attempt.",
      };
    if ((e.bestCorrect ?? 0) >= 3)
      return {
        label: "Solid",
        color:
          "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria: "Solid: named 3 or 4 of 5 values correctly.",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria: "In progress: named fewer than 3 values correctly.",
    };
  }
  if (surface === "review-ai") {
    const p = pct(e.bestCorrect, e.bestTotal);
    if (p >= 90)
      return {
        label: "Mastered",
        color:
          "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
        criteria: "Mastered: best round ≥ 90% first-try correct.",
      };
    if (p >= 70)
      return {
        label: "Solid",
        color:
          "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria: "Solid: best round ≥ 70% first-try correct.",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria: "In progress: best round below 70% first-try correct.",
    };
  }
  if (surface === "tier") {
    const p = pct(e.bestCorrect, e.bestTotal);
    if (p >= 90)
      return {
        label: "Mastered",
        color:
          "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
        criteria: "Mastered: best run ≥ 90% correct.",
      };
    if (p >= 70)
      return {
        label: "Solid",
        color:
          "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
        criteria: "Solid: best run ≥ 70% correct (below 90% is Solid, below 70% is In progress).",
      };
    return {
      label: "In progress",
      color:
        "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
      criteria: "In progress: best run below 70% correct.",
    };
  }
  const p = pct(e.bestCorrect, e.bestTotal);
  const total = e.bestTotal;
  // Minimum-answered floors only kick in when the bucket itself is big enough
  // to meet them. A 6-question pool can never reach 15, so requiring it would
  // trap a perfect scorer at "In progress" forever.
  const masteredFloor = Math.min(15, bucketQuestionCount);
  const solidFloor = Math.min(10, bucketQuestionCount);
  if (p >= 90 && total >= masteredFloor)
    return {
      label: "Mastered",
      color:
        "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
      criteria: `Mastered: best run ≥ 90% correct AND at least ${masteredFloor} questions answered.`,
    };
  if (p >= 70 && total >= solidFloor)
    return {
      label: "Solid",
      color:
        "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200",
      criteria: `Solid: best run ≥ 70% correct AND at least ${solidFloor} questions answered.`,
    };
  return {
    label: "In progress",
    color:
      "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
    criteria:
      `In progress: taken but hasn't yet hit the Solid threshold (≥ 70% AND ≥ ${solidFloor} questions).`,
  };
}

export default function SelfAssessmentsDashboard() {
  const buckets = useMemo(() => allBuckets(), []);
  const ssr = typeof window === "undefined";
  const [glossary] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("glossary"),
  );
  const [arch] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("architecture"),
  );
  const [tracer] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("tracer"),
  );
  const [tier] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("tier"),
  );
  const [matching] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("matching"),
  );
  const [reviewAI] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("review-ai"),
  );
  const [values] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("values"),
  );
  const [flow] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("flow"),
  );
  const [cron] = useState<Record<string, StoredEntry>>(() =>
    ssr ? {} : readSurface("cron"),
  );
  const [formatFilter, setFormatFilter] = useState<QuizFormat | "all">("all");

  const results: Record<SurfaceKind, Record<string, StoredEntry>> = {
    glossary,
    architecture: arch,
    tracer,
    tier,
    matching,
    "review-ai": reviewAI,
    values,
    flow,
    cron,
  };

  const taken = buckets.filter(
    (b) => results[b.surface][b.slug]?.bestCorrect,
  ).length;
  const total = buckets.length;

  // Count how many buckets each format has, so we can show counts in the chips.
  const formatCounts = useMemo(() => {
    const counts: Record<QuizFormat, number> = {
      "adaptive-multiple-choice": 0,
      binary: 0,
      sorting: 0,
      matching: 0,
      "diff-click": 0,
      recall: 0,
    };
    for (const b of buckets) counts[SURFACE_TO_FORMAT[b.surface]] += 1;
    return counts;
  }, [buckets]);

  const ALL_FORMATS: QuizFormat[] = [
    "adaptive-multiple-choice",
    "sorting",
    "binary",
    "matching",
    "diff-click",
    "recall",
  ];

  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-6 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Self-assessments
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Every quiz in Fleet Atlas in one place. {taken} of {total} started.
          Scores persist locally.
        </p>
      </header>

      <div className="mb-8 flex flex-wrap gap-2">
        <button
          onClick={() => setFormatFilter("all")}
          className={`rounded border px-2.5 py-1 text-xs font-semibold transition-colors ${
            formatFilter === "all"
              ? "border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
              : "border-zinc-300 bg-white text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          }`}
        >
          All ({total})
        </button>
        {ALL_FORMATS.filter((f) => formatCounts[f] > 0).map((f) => {
          const active = formatFilter === f;
          return (
            <button
              key={f}
              onClick={() => setFormatFilter(active ? "all" : f)}
              className={`inline-flex items-center gap-1 rounded border px-2.5 py-1 text-xs font-semibold transition-colors ${
                active
                  ? `${QUIZ_FORMAT_COLOR[f]} ring-1 ring-zinc-900 dark:ring-zinc-100`
                  : QUIZ_FORMAT_COLOR[f] + " hover:brightness-105"
              }`}
            >
              {QUIZ_FORMAT_LABEL[f]} ({formatCounts[f]})
            </button>
          );
        })}
      </div>

      {(() => {
        // Each self-assessment bucket is grouped by the left-nav destination
        // whose content it actually tests. Section headers match the nav
        // labels so the page reads like an index of "prove you know this
        // nav item."
        type NavSection = { id: string; label: string; href: string };
        const NAV_SECTIONS: NavSection[] = [
          { id: "architecture", label: "Architecture map", href: "/architecture" },
          { id: "subsystems", label: "Subsystems", href: "/subsystems" },
          { id: "tracer", label: "Request tracer", href: "/tracer" },
          { id: "background", label: "Background work", href: "/background" },
          { id: "glossary", label: "Glossary", href: "/glossary" },
          { id: "tiers", label: "Free vs Premium", href: "/tiers" },
          { id: "backend-conventions", label: "Backend conventions", href: "/backend-conventions" },
          { id: "handbook", label: "Fleet handbook bridge", href: "/handbook" },
        ];

        function navSectionFor(
          surface: SurfaceKind,
          slug: string,
        ): string {
          if (surface === "architecture") return "architecture";
          if (surface === "tracer") return "tracer";
          if (surface === "flow") return "subsystems";
          if (surface === "glossary") return "glossary";
          if (surface === "matching") {
            if (slug === "osquery-tls" || slug === "webhook-types") {
              return "subsystems";
            }
            if (slug === "interval-defaults" || slug === "queue-storage") {
              return "background";
            }
            return "glossary";
          }
          if (surface === "tier") return "tiers";
          if (surface === "review-ai") return "backend-conventions";
          if (surface === "values") return "handbook";
          if (surface === "cron") return "background";
          return "glossary";
        }

        // Pre-group buckets by nav section, honoring the format filter.
        const grouped: Record<string, UnifiedBucket[]> = {};
        for (const b of buckets) {
          if (
            formatFilter !== "all" &&
            SURFACE_TO_FORMAT[b.surface] !== formatFilter
          ) {
            continue;
          }
          const sectionId = navSectionFor(b.surface, b.slug);
          if (!grouped[sectionId]) grouped[sectionId] = [];
          grouped[sectionId].push(b);
        }

        // Render only sections that actually have buckets visible.
        const visibleSections = NAV_SECTIONS.filter(
          (s) => (grouped[s.id]?.length ?? 0) > 0,
        );

        return visibleSections.map((section) => {
          return (
            <section key={section.id} className="mb-10">
              <div className="mb-3">
                <Link
                  href={section.href}
                  className="inline-block text-sm font-semibold uppercase tracking-wide text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200"
                >
                  {section.label}
                </Link>
              </div>
              <ul className="grid gap-3 md:grid-cols-2">
                {grouped[section.id].map((b) => {
                  const surface = b.surface;
                  const e = results[surface][b.slug];
                  const status = statusOf(e, surface, b.questionCount);
                  const cardHref =
                    surface === "tracer"
                      ? `/tracer/${b.slug}`
                      : surface === "tier"
                        ? "/tiers#quiz"
                        : surface === "matching"
                          ? b.surfaceHref
                          : surface === "review-ai"
                            ? "/review-ai"
                            : surface === "values"
                              ? "/values"
                              : surface === "flow"
                                ? b.surfaceHref
                                : surface === "cron"
                                  ? b.surfaceHref
                                  : `${b.surfaceHref}?bucket=${encodeURIComponent(b.slug)}`;
              return (
                <li key={`${surface}:${b.slug}`}>
                  <Link
                    href={cardHref}
                    className="flex h-full flex-col rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                  >
                    <div className="mb-1 flex items-start justify-between gap-3">
                      <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                        {b.label}
                      </h3>
                      <div className="shrink-0">
                        <QuizFormatChip format={SURFACE_TO_FORMAT[surface]} />
                      </div>
                    </div>
                    <p className="text-xs text-zinc-600 dark:text-zinc-400">
                      {b.description}
                    </p>
                    <div className="mt-auto flex items-end justify-between gap-3 pt-3">
                      <div className="flex flex-wrap items-center gap-2 text-xs">
                        <span className="text-zinc-500">
                          {b.questionCount} questions
                        </span>
                        {e?.bestCorrect ? (
                          <>
                            <span className="text-zinc-400">·</span>
                            <span className="text-zinc-700 dark:text-zinc-300">
                              <span className="font-semibold text-emerald-700 dark:text-emerald-300">
                                Best
                              </span>{" "}
                              {e.bestCorrect}/{e.bestTotal} (
                              {pct(e.bestCorrect, e.bestTotal)}%)
                            </span>
                            {e.correct && e.correct !== e.bestCorrect ? (
                              <>
                                <span className="text-zinc-400">·</span>
                                <span className="text-zinc-600 dark:text-zinc-400">
                                  Last {e.correct}/{e.total}
                                </span>
                              </>
                            ) : null}
                          </>
                        ) : null}
                      </div>
                      <span
                        title={status.criteria}
                        className={`shrink-0 cursor-help rounded border px-2 py-0.5 text-[11px] font-semibold ${status.color}`}
                      >
                        {status.label}
                      </span>
                    </div>
                  </Link>
                </li>
              );
            })}
          </ul>
        </section>
          );
        });
      })()}
    </div>
  );
}
