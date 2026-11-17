"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { filterItems, PALETTE_ITEMS, type PaletteItem } from "./palette-items";
import { usePalette } from "./PaletteContext";

const GROUP_ORDER: PaletteItem["group"][] = [
  "Modes",
  "Start here",
  "Architecture",
  "Tracers",
  "Data layer",
  "Background",
  "Subsystems",
  "Scenarios",
  "Glossary",
  "Backend conventions",
];

const LISTBOX_ID = "fleet-atlas-palette-listbox";
const optionId = (i: number) => `fleet-atlas-palette-option-${i}`;

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

export default function CommandPalette() {
  const router = useRouter();
  const { open, setOpen } = usePalette();
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);

  const results = useMemo(() => filterItems(query), [query]);

  const grouped = useMemo(() => {
    const map: Record<string, PaletteItem[]> = {};
    for (const g of GROUP_ORDER) map[g] = [];
    for (const it of results) map[it.group].push(it);
    return GROUP_ORDER.map((g) => ({ group: g, items: map[g] })).filter(
      (g) => g.items.length > 0,
    );
  }, [results]);

  const close = useCallback(() => {
    setOpen(false);
    setQuery("");
    setSelected(0);
  }, [setOpen]);

  const openIt = useCallback(() => {
    setOpen(true);
    setSelected(0);
  }, [setOpen]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const isCmdK =
        (e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k";
      if (isCmdK) {
        // Only hijack Cmd+K from an editable target when the palette is
        // already open (so its own input can still toggle it closed).
        if (!open && isEditableTarget(e.target)) return;
        e.preventDefault();
        if (open) close();
        else openIt();
      } else if (open && e.key === "Escape") {
        e.preventDefault();
        close();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, openIt, close]);

  useEffect(() => {
    if (open) {
      previouslyFocusedRef.current =
        document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null;
      requestAnimationFrame(() => inputRef.current?.focus());
      return;
    }
    const prev = previouslyFocusedRef.current;
    previouslyFocusedRef.current = null;
    prev?.focus();
  }, [open]);


  const navigate = (href: string) => {
    close();
    router.push(href);
  };

  const onInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelected((s) => Math.min(s + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelected((s) => Math.max(s - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const target = results[selected];
      if (target) navigate(target.href);
    } else if (e.key === "Tab") {
      // Trap Tab inside the palette; arrow keys handle option navigation.
      e.preventDefault();
    }
  };

  useEffect(() => {
    if (!open) return;
    const el = listRef.current?.querySelector<HTMLElement>(
      '[data-selected="true"]',
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [selected, open]);

  if (!open) return null;

  let runningIndex = -1;

  const activeDescendantId =
    results.length > 0 && selected >= 0 && selected < results.length
      ? optionId(selected)
      : undefined;

  return (
    <div
      className="fixed inset-0 z-[999] flex items-start justify-center bg-black/40 pt-24 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) close();
      }}
    >
      <div className="w-full max-w-xl overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-2xl dark:border-zinc-700 dark:bg-zinc-900">
        <div className="flex items-center gap-2 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            className="text-zinc-500"
            aria-hidden="true"
          >
            <circle cx="11" cy="11" r="7" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <input
            ref={inputRef}
            data-no-focus-outline
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelected(0);
            }}
            onKeyDown={onInputKeyDown}
            placeholder="Jump to a node, tracer, subsystem, scenario..."
            className="flex-1 bg-transparent text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none dark:text-zinc-100 dark:placeholder:text-zinc-500"
            role="combobox"
            aria-expanded="true"
            aria-controls={LISTBOX_ID}
            aria-autocomplete="list"
            aria-activedescendant={activeDescendantId}
            aria-label="Search destinations"
          />
          <kbd className="rounded border border-zinc-300 bg-zinc-100 px-1.5 py-0.5 text-[10px] font-mono text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400">
            esc
          </kbd>
        </div>
        <div
          ref={listRef}
          id={LISTBOX_ID}
          role="listbox"
          aria-label="Destinations"
          className="max-h-[60vh] overflow-y-auto py-2"
        >
          {results.length === 0 ? (
            <div className="px-4 py-6 text-center text-sm text-zinc-500">
              No matches.
            </div>
          ) : (
            grouped.map(({ group, items }) => (
              <div key={group} role="presentation" className="mb-2">
                <div
                  role="presentation"
                  className="px-4 py-1 text-[10px] font-semibold uppercase tracking-wide text-zinc-500"
                >
                  {group}
                </div>
                {items.map((it) => {
                  runningIndex += 1;
                  const isSel = runningIndex === selected;
                  return (
                    <button
                      key={`${it.group}-${it.href}`}
                      id={optionId(runningIndex)}
                      role="option"
                      aria-selected={isSel}
                      data-selected={isSel}
                      tabIndex={-1}
                      onMouseEnter={() => setSelected(runningIndex)}
                      onClick={() => navigate(it.href)}
                      className={`flex w-full items-start gap-3 px-4 py-2 text-left ${
                        isSel
                          ? "bg-sky-50 dark:bg-sky-950/40"
                          : "hover:bg-zinc-50 dark:hover:bg-zinc-800/40"
                      }`}
                    >
                      <div className="flex-1 overflow-hidden">
                        <div className="truncate text-sm text-zinc-900 dark:text-zinc-100">
                          {it.label}
                        </div>
                        {it.hint ? (
                          <div className="truncate text-xs text-zinc-500">
                            {it.hint}
                          </div>
                        ) : null}
                      </div>
                      {isSel ? (
                        <kbd className="shrink-0 rounded border border-zinc-300 bg-zinc-100 px-1.5 py-0.5 text-[10px] font-mono text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400">
                          ↵
                        </kbd>
                      ) : null}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>
        <div className="flex items-center justify-between border-t border-zinc-200 bg-zinc-50 px-4 py-2 text-[11px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-950">
          <span>{PALETTE_ITEMS.length} destinations</span>
          <span className="flex items-center gap-2">
            <kbd className="rounded border border-zinc-300 bg-white px-1 py-0.5 font-mono text-[10px] dark:border-zinc-700 dark:bg-zinc-900">
              ↑↓
            </kbd>
            navigate
            <kbd className="rounded border border-zinc-300 bg-white px-1 py-0.5 font-mono text-[10px] dark:border-zinc-700 dark:bg-zinc-900">
              ↵
            </kbd>
            open
          </span>
        </div>
      </div>
    </div>
  );
}
