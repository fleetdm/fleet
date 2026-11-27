"use client";

import { usePalette } from "./PaletteContext";

export default function SearchButton() {
  const { setOpen } = usePalette();
  return (
    <button
      onClick={() => setOpen(true)}
      className="flex w-full items-center justify-between gap-2 rounded border border-zinc-300 bg-white px-2 py-1.5 text-xs text-zinc-500 hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:border-zinc-600 dark:hover:bg-zinc-800"
      aria-label="Search (Cmd+K or Ctrl+K)"
      aria-keyshortcuts="Meta+K Control+K"
      title="Search (⌘K / Ctrl+K)"
    >
      <span className="flex items-center gap-2">
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <circle cx="11" cy="11" r="7" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        Search
      </span>
      <kbd className="rounded border border-zinc-300 bg-zinc-50 px-1 font-mono text-[10px] text-zinc-600 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-400">
        ⌘K
      </kbd>
    </button>
  );
}
