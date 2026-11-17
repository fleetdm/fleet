"use client";

import Link from "next/link";
import { useState } from "react";
import {
  PATHS,
  TEAMS,
  type PathKind,
  type RolePath,
} from "@/components/start/paths";
import AssessmentPanel from "@/components/start/AssessmentPanel";
import {
  CROSS_CUTTING,
  handbookRoleForSlug,
} from "@/components/start/handbook-data";
import { useIsHydrated } from "@/lib/use-is-hydrated";

const AXIS_KEY = "fleet-atlas:start-axis";
const ROLE_KEY = "fleet-atlas:start-role";
const TEAM_KEY = "fleet-atlas:start-team";
const VIEW_KEY = "fleet-atlas:start-view";

type View = "atlas" | "handbook";

export default function StartPage() {
  const hydrated = useIsHydrated();
  const [axis, setAxis] = useState<PathKind>(() => {
    if (typeof window === "undefined") return "role";
    const stored = localStorage.getItem(AXIS_KEY);
    return stored === "role" || stored === "team" ? stored : "role";
  });
  const [role, setRole] = useState<string>(() => {
    if (typeof window === "undefined") return PATHS[0].slug;
    const stored = localStorage.getItem(ROLE_KEY);
    return stored && PATHS.some((p) => p.slug === stored)
      ? stored
      : PATHS[0].slug;
  });
  const [team, setTeam] = useState<string>(() => {
    if (typeof window === "undefined") return TEAMS[0].slug;
    const stored = localStorage.getItem(TEAM_KEY);
    return stored && TEAMS.some((p) => p.slug === stored)
      ? stored
      : TEAMS[0].slug;
  });
  const [view, setView] = useState<View>(() => {
    if (typeof window === "undefined") return "atlas";
    const stored = localStorage.getItem(VIEW_KEY);
    return stored === "atlas" || stored === "handbook" ? stored : "atlas";
  });

  const chooseView = (v: View) => {
    setView(v);
    localStorage.setItem(VIEW_KEY, v);
  };

  const chooseAxis = (a: PathKind) => {
    setAxis(a);
    localStorage.setItem(AXIS_KEY, a);
  };

  const chooseRole = (slug: string) => {
    setRole(slug);
    localStorage.setItem(ROLE_KEY, slug);
  };

  const chooseTeam = (slug: string) => {
    setTeam(slug);
    localStorage.setItem(TEAM_KEY, slug);
  };

  const options: RolePath[] = axis === "role" ? PATHS : TEAMS;
  const activeSlug = axis === "role" ? role : team;
  const chooser = axis === "role" ? chooseRole : chooseTeam;
  const path = options.find((p) => p.slug === activeSlug) ?? options[0];

  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <header className="mb-6 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Start here
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Pick a path by your role, or by your team. Your choice sticks across
          visits.
        </p>
      </header>

      <div className="mb-6 inline-flex items-center gap-1 rounded border border-zinc-200 bg-white p-1 text-sm dark:border-zinc-800 dark:bg-zinc-950">
        {(["role", "team"] as const).map((a) => (
          <button
            key={a}
            onClick={() => chooseAxis(a)}
            className={`rounded px-3 py-1 ${
              axis === a
                ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
            }`}
          >
            {a === "role" ? "By role" : "By team"}
          </button>
        ))}
      </div>

      <fieldset className="mb-8 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <legend className="px-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
          {axis === "role" ? "Your role" : "Your team"}
        </legend>
        <div className="grid gap-2 sm:grid-cols-2">
          {options.map((p) => {
            const checked = activeSlug === p.slug;
            return (
              <label
                key={p.slug}
                className={`flex cursor-pointer items-start gap-3 rounded border p-3 transition-colors ${
                  checked
                    ? "border-zinc-900 bg-zinc-100 dark:border-zinc-100 dark:bg-zinc-800"
                    : "border-zinc-200 bg-white hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                }`}
              >
                <input
                  type="radio"
                  name={axis}
                  value={p.slug}
                  checked={checked}
                  onChange={() => chooser(p.slug)}
                  className="mt-1 h-4 w-4 accent-zinc-900 dark:accent-zinc-100"
                />
                <span className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                  {p.label}
                </span>
              </label>
            );
          })}
        </div>
      </fieldset>

      {hydrated ? (
        <section
          key={`${axis}-${path.slug}`}
          id={path.slug}
          className="rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950"
        >
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            {path.label}
          </h2>
          <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
            {path.intro}
          </p>

          <div className="mt-5 inline-flex items-center gap-1 rounded border border-zinc-200 bg-white p-1 text-xs dark:border-zinc-800 dark:bg-zinc-950">
            {(["atlas", "handbook"] as const).map((v) => (
              <button
                key={v}
                onClick={() => chooseView(v)}
                className={`cursor-pointer rounded px-3 py-1 ${
                  view === v
                    ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                    : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
                }`}
              >
                {v === "atlas" ? "Atlas path" : "Fleet handbook"}
              </button>
            ))}
          </div>

          {view === "atlas" ? (
            <>
              <ol className="mt-5 space-y-3">
                {path.steps.map((s, i) => (
                  <li key={i}>
                    <Link
                      href={s.href}
                      className="group flex gap-3 rounded border border-zinc-200 p-3 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                    >
                      <span className="w-6 shrink-0 pt-0.5 text-right font-mono text-xs text-zinc-500">
                        {String(i + 1).padStart(2, "0")}
                      </span>
                      <div className="flex-1">
                        <span className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                          {s.label}
                        </span>
                        <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                          {s.why}
                        </p>
                      </div>
                    </Link>
                  </li>
                ))}
              </ol>

              <AssessmentPanel assessments={path.assessments} />
            </>
          ) : (
            <HandbookView slug={path.slug} />
          )}
        </section>
      ) : null}
    </div>
  );
}

function HandbookView({ slug }: { slug: string }) {
  const roleHandbook = handbookRoleForSlug(slug);
  return (
    <div className="mt-5 space-y-6">
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Everyone should read these
        </h3>
        <ul className="mt-3 grid gap-3 md:grid-cols-2">
          {CROSS_CUTTING.map((h) => (
            <li key={h.href}>
              <a
                href={h.href}
                target="_blank"
                rel="noopener"
                className="block h-full rounded border border-zinc-200 bg-white p-3 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
              >
                <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                  {h.title} ↗
                </div>
                <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                  {h.blurb}
                </p>
              </a>
            </li>
          ))}
        </ul>
      </div>

      {roleHandbook ? (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
            {roleHandbook.roleLabel}
          </h3>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            {roleHandbook.intro}
          </p>
          <ul className="mt-3 space-y-2">
            {roleHandbook.pointers.map((h) => (
              <li key={h.href}>
                <a
                  href={h.href}
                  target="_blank"
                  rel="noopener"
                  className="block rounded border border-zinc-200 bg-white p-3 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
                >
                  <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                    {h.title} ↗
                  </div>
                  <p className="mt-0.5 text-xs text-zinc-600 dark:text-zinc-400">
                    {h.blurb}
                  </p>
                </a>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
