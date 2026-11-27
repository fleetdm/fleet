"use client";

import { useEffect, useState } from "react";
import Markdown from "@/components/Markdown";
import { ACTOR_COLORS, type Scenario } from "./scenarios";
import FlowView from "./FlowView";
import { editorLink } from "@/lib/config";

export default function ScenarioBody({ scenario }: { scenario: Scenario }) {
  const [flowOpen, setFlowOpen] = useState(false);

  useEffect(() => {
    if (!flowOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setFlowOpen(false);
    };
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", onKey);
    };
  }, [flowOpen]);

  return (
    <>
      <section className="mb-6 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
        <Markdown>{scenario.overview}</Markdown>
      </section>

      <div className="mb-4 flex justify-end">
        <button
          onClick={() => setFlowOpen(true)}
          className="flex items-center gap-2 rounded border border-zinc-300 bg-white px-3 py-1.5 text-xs font-medium text-zinc-800 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800"
        >
          <svg
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <path d="M3 3h7v2H5v5H3zm11 0h7v7h-2V5h-5zm7 11v7h-7v-2h5v-5zM10 21H3v-7h2v5h5z" />
          </svg>
          View as flow chart
        </button>
      </div>

      <ol className="space-y-3">
        {scenario.steps.map((step, i) => (
          <li
            key={i}
            className="flex gap-4 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950"
          >
            <div className="w-24 shrink-0 font-mono text-xs text-zinc-500">
              {step.t}
            </div>
            <div className="flex-1">
              <div className="mb-1 flex items-center gap-2">
                <span
                  className={`rounded px-2 py-0.5 text-[11px] font-semibold ${
                    ACTOR_COLORS[step.actor] ?? ""
                  }`}
                >
                  {step.actor}
                </span>
              </div>
              <div className="text-sm text-zinc-900 dark:text-zinc-100">
                {step.action}
              </div>
              {step.detail ? (
                <div className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                  {step.detail}
                </div>
              ) : null}
              {step.file ? (
                <a
                  href={editorLink(step.file)}
                  className="mt-1 inline-block font-mono text-xs text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-300"
                >
                  {step.file}
                </a>
              ) : null}
            </div>
          </li>
        ))}
      </ol>

      <section className="mt-10 rounded border border-amber-300 bg-amber-50 p-5 dark:border-amber-800 dark:bg-amber-950/40">
        <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
          Takeaway
        </div>
        <p className="text-sm text-amber-900 dark:text-amber-100">
          {scenario.takeaway}
        </p>
      </section>

      {flowOpen ? (
        <div
          className="fixed inset-0 z-[999] flex flex-col bg-black/40 backdrop-blur-sm"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setFlowOpen(false);
          }}
        >
          <div className="m-4 flex flex-1 flex-col overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-2xl dark:border-zinc-700 dark:bg-zinc-900">
            <header className="flex items-center justify-between gap-3 border-b border-zinc-200 px-5 py-3 dark:border-zinc-800">
              <div>
                <h2 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                  {scenario.label}
                </h2>
                <p className="text-xs text-zinc-500">
                  Flow chart. Each column is one actor; arrows cross lanes when
                  control passes.
                </p>
              </div>
              <button
                onClick={() => setFlowOpen(false)}
                className="rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                title="Close (Esc)"
              >
                Close
              </button>
            </header>
            <div className="flex-1">
              <FlowView steps={scenario.steps} />
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
