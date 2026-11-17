"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { TechnicalHelpToggle } from "./TechnicalHelp";
import { ThemeToggle } from "./Theme";
import SearchButton from "./palette/SearchButton";

// Nav items are grouped by atlas-style purpose. Group boundaries are
// signaled by a hairline divider alone. No labels, no colored dots.
// Reader pattern-recognition picks up the grouping from the whitespace.
type NavItem = { href: string; label: string };
type NavGroup = {
  kind: "orientation" | "maps" | "exercises" | "gazetteer";
  items: NavItem[];
};

const GROUPS: NavGroup[] = [
  {
    kind: "orientation",
    items: [
      { href: "/start", label: "Start here" },
      { href: "/dev-setup", label: "Dev setup" },
    ],
  },
  {
    kind: "maps",
    items: [
      { href: "/architecture", label: "Architecture map" },
      { href: "/subsystems", label: "Subsystems" },
      { href: "/tracer", label: "Request tracer" },
      { href: "/scenarios", label: "Scenarios" },
      { href: "/data", label: "Data layer" },
      { href: "/background", label: "Background work" },
    ],
  },
  {
    kind: "exercises",
    items: [{ href: "/self-assessments", label: "Self-assessments" }],
  },
  {
    kind: "gazetteer",
    items: [
      { href: "/glossary", label: "Glossary" },
      { href: "/tiers", label: "Free vs Premium" },
      { href: "/backend-conventions", label: "Backend conventions" },
      { href: "/handbook", label: "Fleet handbook bridge" },
    ],
  },
];

export default function Nav() {
  const pathname = usePathname();
  return (
    <nav className="sticky top-0 flex h-screen w-64 flex-col border-r border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
      <div className="mb-6">
        <div className="flex items-center gap-2">
          <Image
            src="/atlas-logo.svg"
            alt=""
            width={14}
            height={14}
            className="shrink-0"
          />
          <div className="text-base font-semibold leading-none text-zinc-900 dark:text-zinc-100">
            Fleet Atlas
          </div>
        </div>
        <div className="mt-2 text-xs text-zinc-500">
          Local maps, scenarios, and quizzes for learning Fleet
        </div>
      </div>

      <div className="space-y-3">
        {GROUPS.map((group, groupIdx) => (
          <div key={group.kind}>
            {groupIdx > 0 ? (
              <div className="mx-3 mb-3 border-t border-zinc-200 dark:border-zinc-800" />
            ) : null}
            <ul className="space-y-1">
              {group.items.map((item) => {
                const active =
                  item.href === "/"
                    ? pathname === "/"
                    : pathname.startsWith(item.href);
                const isExercises = group.kind === "exercises";
                const stateClasses = isExercises
                  ? active
                    ? "bg-emerald-50 text-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-200"
                    : "text-emerald-700 hover:bg-emerald-50 hover:text-emerald-900 dark:text-emerald-500 dark:hover:bg-emerald-950/40 dark:hover:text-emerald-300"
                  : active
                    ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-50"
                    : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-900 dark:hover:text-zinc-200";
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      className={`block rounded px-3 py-2 text-sm ${stateClasses}`}
                    >
                      {item.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>

      <div className="mt-3 px-1">
        <SearchButton />
      </div>
      <div className="flex-1" />
      <div className="flex items-center gap-2 pt-4">
        <TechnicalHelpToggle />
        <ThemeToggle />
      </div>
    </nav>
  );
}
