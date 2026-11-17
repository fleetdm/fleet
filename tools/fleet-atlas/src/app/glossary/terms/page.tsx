"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import {
  CATEGORY_COLORS,
  TERMS,
  type GlossaryCategory,
} from "@/components/glossary/terms";
import { BUCKETS } from "@/components/glossary/questions";
import AssessmentPanel from "@/components/start/AssessmentPanel";
import type { AssessmentRef } from "@/components/start/paths";
import { TechnicalHelp } from "@/components/TechnicalHelp";

const ALL_GLOSSARY_ASSESSMENTS: AssessmentRef[] = BUCKETS.map((b) => ({
  surface: "glossary" as const,
  bucketSlug: b.slug,
  bucketLabel: `${b.label} (glossary)`,
}));

export default function GlossaryTermsPage() {
  const [query, setQuery] = useState("");
  const [activeCategory, setActiveCategory] = useState<GlossaryCategory | null>(
    null,
  );

  const categories = useMemo(() => {
    const set = new Set<GlossaryCategory>();
    for (const t of TERMS) set.add(t.category);
    return Array.from(set);
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return TERMS.filter((t) => {
      if (activeCategory && t.category !== activeCategory) return false;
      if (!q) return true;
      const hay = `${t.term} ${(t.aka ?? []).join(" ")} ${t.short} ${t.long ?? ""}`.toLowerCase();
      return q.split(/\s+/).every((tok) => hay.includes(tok));
    });
  }, [query, activeCategory]);

  const bySlug = useMemo(() => {
    const m: Record<string, (typeof TERMS)[number]> = {};
    for (const t of TERMS) m[t.slug] = t;
    return m;
  }, []);

  return (
    <>
      <div className="mb-6 space-y-3">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search terms, acronyms, or descriptions..."
          className="w-full rounded border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-zinc-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus:border-zinc-500"
        />
        <div className="flex flex-wrap gap-2 text-xs">
          <button
            onClick={() => setActiveCategory(null)}
            className={`rounded border px-2 py-0.5 ${
              activeCategory === null
                ? "border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
                : "border-zinc-300 bg-white text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
            }`}
          >
            All ({TERMS.length})
          </button>
          {categories.map((c) => {
            const count = TERMS.filter((t) => t.category === c).length;
            const active = activeCategory === c;
            return (
              <button
                key={c}
                onClick={() => setActiveCategory(active ? null : c)}
                className={`inline-flex items-center gap-1 rounded px-2 py-0.5 ${CATEGORY_COLORS[c]} ${
                  active ? "ring-1 ring-zinc-900 dark:ring-zinc-100" : ""
                }`}
              >
                {c} <span className="opacity-60">({count})</span>
              </button>
            );
          })}
        </div>
      </div>

      {visible.length === 0 ? (
        <p className="text-sm text-zinc-500">No matches.</p>
      ) : (
        <ul className="grid gap-3 md:grid-cols-2">
          {visible.map((t) => (
            <li
              key={t.slug}
              id={t.slug}
              className="scroll-mt-6 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950"
            >
              <div className="flex items-baseline gap-2">
                <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                  {t.term}
                </h3>
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${CATEGORY_COLORS[t.category]}`}
                >
                  {t.category}
                </span>
              </div>
              {t.aka && t.aka.length > 0 ? (
                <div className="mt-1 text-xs text-zinc-500">
                  aka {t.aka.join(" / ")}
                </div>
              ) : null}
              <p className="mt-2 text-sm text-zinc-800 dark:text-zinc-200">
                {t.short}
              </p>
              {t.long ? (
                <p className="mt-2 text-xs text-zinc-600 dark:text-zinc-400">
                  {t.long}
                </p>
              ) : null}
              {t.technicalHelp ? (
                <div className="mt-2">
                  <TechnicalHelp
                    technical={t.technicalHelp.technical}
                    analogies={t.technicalHelp.analogies}
                  />
                </div>
              ) : null}
              {(t.seeAlso && t.seeAlso.length > 0) ||
              (t.links && t.links.length > 0) ? (
                <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
                  {t.seeAlso?.map((slug) => {
                    const target = bySlug[slug];
                    if (!target) return null;
                    return (
                      <a
                        key={slug}
                        href={`#${slug}`}
                        className="rounded border border-zinc-300 bg-white px-2 py-0.5 text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
                      >
                        → {target.term}
                      </a>
                    );
                  })}
                  {t.links?.map((l) => (
                    <Link
                      key={l.href}
                      href={l.href}
                      className="rounded border border-zinc-300 bg-zinc-50 px-2 py-0.5 text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                    >
                      {l.label}
                    </Link>
                  ))}
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      <AssessmentPanel assessments={ALL_GLOSSARY_ASSESSMENTS} />
    </>
  );
}
