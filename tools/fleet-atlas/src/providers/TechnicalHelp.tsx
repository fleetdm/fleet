"use client";

import { createContext, useContext, useEffect, useRef, useState } from "react";

type TechnicalHelpCtx = {
  on: boolean;
  toggle: () => void;
};

const Ctx = createContext<TechnicalHelpCtx | null>(null);

const KEY = "fleet-atlas:technical-help-on";

export function TechnicalHelpProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [on, setOn] = useState<boolean>(() => {
    if (typeof window === "undefined") return true;
    const stored = localStorage.getItem(KEY);
    return stored == null ? true : stored === "1";
  });
  const toggle = () => {
    setOn((prev) => {
      const next = !prev;
      localStorage.setItem(KEY, next ? "1" : "0");
      return next;
    });
  };
  return <Ctx.Provider value={{ on, toggle }}>{children}</Ctx.Provider>;
}

export function useTechnicalHelp() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useTechnicalHelp outside provider");
  return ctx;
}

export function TechnicalHelpToggle() {
  const { on, toggle } = useTechnicalHelp();
  return (
    <button
      onClick={toggle}
      aria-pressed={on}
      className={`flex h-9 items-center justify-center rounded-full border px-3 text-xs font-medium transition-colors ${
        on
          ? "border-sky-300 bg-sky-50 text-sky-800 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200 dark:hover:bg-sky-900/60"
          : "border-zinc-300 bg-white text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
      }`}
      title="Toggle Technical help mode: adds a pop-out helper next to dense content that explains the technical concept in plain terms with analogies."
    >
      Technical help
    </button>
  );
}

type TechnicalHelpProps = {
  // The dense/technical statement. Shown first in the popover.
  technical: string;
  // One or many plain-language analogies.
  analogies?: string[];
};

export function TechnicalHelp({ technical, analogies }: TechnicalHelpProps) {
  const { on } = useTechnicalHelp();
  const [open, setOpen] = useState(false);
  const [hover, setHover] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDocClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [open]);

  if (!on) return null;

  const list: string[] = analogies ?? [];

  const show = open || hover;

  return (
    <div
      ref={ref}
      className="relative my-3 inline-block"
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center gap-1 rounded-full border border-sky-300 bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-800 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200 dark:hover:bg-sky-900/60"
        aria-expanded={show}
      >
        <span aria-hidden>💡</span>
        Technical help
      </button>
      {show ? (
        <div
          className="absolute left-0 top-full z-20 mt-2 w-[32rem] max-w-[90vw] rounded-lg border border-sky-300 bg-white p-4 text-sm text-zinc-900 shadow-xl dark:border-sky-800 dark:bg-zinc-900 dark:text-zinc-100"
          role="dialog"
        >
          <div className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-sky-700 dark:text-sky-400">
            Technical help
          </div>
          <div className="mb-3">
            <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-zinc-500">
              What&apos;s happening (technical)
            </div>
            <div className="text-sm text-zinc-800 dark:text-zinc-200">
              {technical}
            </div>
          </div>
          {list.length > 0 ? (
            <div>
              <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-zinc-500">
                In plain terms
              </div>
              {list.length === 1 ? (
                <div className="text-sm text-zinc-800 dark:text-zinc-200">
                  {list[0]}
                </div>
              ) : (
                <ul className="list-disc space-y-1 pl-5 text-sm text-zinc-800 dark:text-zinc-200">
                  {list.map((a, i) => (
                    <li key={i}>{a}</li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
