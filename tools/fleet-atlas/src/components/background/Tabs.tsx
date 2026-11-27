"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { pathHasEmbeddedQuiz } from "@/lib/embedded-quizzes";

const TABS = [
  { href: "/background/cron", label: "Cron schedules" },
  { href: "/background/workers", label: "Worker queue" },
  { href: "/background/queues", label: "All queues" },
];

export default function BackgroundTabs() {
  const pathname = usePathname();
  return (
    <nav className="mb-8 flex gap-1 border-b border-zinc-200 dark:border-zinc-800">
      {TABS.map((t) => {
        const active = pathname.startsWith(t.href);
        return (
          <Link
            key={t.href}
            href={t.href}
            className={`-mb-px inline-flex items-center gap-1.5 rounded-t border-x border-t px-4 py-2 text-sm ${
              active
                ? "border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100"
                : "border-transparent text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-200"
            }`}
          >
            {t.label}
            {pathHasEmbeddedQuiz(t.href) ? (
              <span
                aria-hidden="true"
                title="This page has an embedded quiz"
                className="text-emerald-600 dark:text-emerald-400"
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M9 11l3 3L22 4" />
                  <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
                </svg>
              </span>
            ) : null}
          </Link>
        );
      })}
    </nav>
  );
}
