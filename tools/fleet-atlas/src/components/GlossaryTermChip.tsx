"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { TERMS, CATEGORY_COLORS } from "@/components/glossary/terms";
import { TechnicalHelp } from "@/components/TechnicalHelp";

export default function GlossaryTermChip({
  slug,
  children,
}: {
  slug: string;
  children?: React.ReactNode;
}) {
  const term = TERMS.find((t) => t.slug === slug);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onEsc);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onEsc);
    };
  }, [open]);

  if (!term) {
    // Unknown slug: render the raw text so the content isn't broken.
    return <>{children}</>;
  }

  return (
    <span ref={ref} className="relative inline-block">
      <button
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="inline-flex items-center gap-1 rounded border border-sky-300 bg-sky-50 px-1.5 py-0.5 text-xs font-medium text-sky-800 no-underline hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200 dark:hover:bg-sky-900/60"
      >
        {children || term.term}
      </button>
      {open && (
        <span
          className="absolute left-0 top-full z-30 mt-2 block w-[26rem] max-w-[90vw] rounded-lg border border-sky-300 bg-white p-4 text-sm text-zinc-900 shadow-xl dark:border-sky-800 dark:bg-zinc-900 dark:text-zinc-100"
          onClick={(e) => e.stopPropagation()}
        >
          <span className="mb-2 flex items-baseline gap-2">
            <span className="text-base font-semibold">{term.term}</span>
            <span
              className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${CATEGORY_COLORS[term.category]}`}
            >
              {term.category}
            </span>
          </span>
          {term.aka && term.aka.length > 0 ? (
            <span className="mt-1 block text-xs text-zinc-500">
              aka {term.aka.join(" / ")}
            </span>
          ) : null}
          <span className="mt-2 block text-sm text-zinc-800 dark:text-zinc-200">
            {term.short}
          </span>
          {term.long ? (
            <span className="mt-2 block text-xs text-zinc-600 dark:text-zinc-400">
              {term.long}
            </span>
          ) : null}
          {term.technicalHelp ? (
            <span className="mt-2 block">
              <TechnicalHelp
                technical={term.technicalHelp.technical}
                analogies={term.technicalHelp.analogies}
              />
            </span>
          ) : null}
          <span className="mt-3 block">
            <Link
              href={`/glossary/terms#${term.slug}`}
              target="_blank"
              rel="noopener"
              className="text-xs text-zinc-500 underline hover:text-zinc-800 dark:hover:text-zinc-200"
            >
              Open in glossary ↗
            </Link>
          </span>
        </span>
      )}
    </span>
  );
}
