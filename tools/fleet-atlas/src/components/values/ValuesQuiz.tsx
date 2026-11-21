"use client";

import { useState } from "react";
import { FLEET_VALUES, canonicalize } from "./values";
import { useIsHydrated } from "@/hooks/use-is-hydrated";

const STORAGE_KEY = "fleet-atlas:values-quiz-best";

type Best = {
  bestCorrect: number;
  bestTotal: number;
  takenAt: number;
};

function readBest(): Best | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Best;
  } catch {
    return null;
  }
}

function writeBestIfBetter(correct: number, total: number) {
  if (typeof window === "undefined") return;
  const current = readBest();
  if (!current || correct > current.bestCorrect) {
    const next: Best = { bestCorrect: correct, bestTotal: total, takenAt: Date.now() };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  }
}

type Result = {
  // Per-input state: did this specific input match a Fleet value?
  // Note: correctness is per unique value, so if the user types the same
  // value in two inputs, only one counts. We track which canonical names
  // have been claimed so the second duplicate reads as wrong.
  perInput: Array<{ canonical: string | null; counted: boolean }>;
  correctCount: number;
};

function scoreAnswers(answers: string[]): Result {
  const claimed = new Set<string>();
  const perInput = answers.map((a) => {
    const canonical = canonicalize(a);
    if (canonical && !claimed.has(canonical)) {
      claimed.add(canonical);
      return { canonical, counted: true };
    }
    return { canonical, counted: false };
  });
  return { perInput, correctCount: claimed.size };
}

export default function ValuesQuiz() {
  const hydrated = useIsHydrated();
  const [answers, setAnswers] = useState<string[]>(["", "", "", "", ""]);
  const [submitted, setSubmitted] = useState(false);
  const [best, setBest] = useState<Best | null>(() => readBest());

  const result = submitted ? scoreAnswers(answers) : null;

  function updateAnswer(i: number, next: string) {
    setAnswers((prev) => {
      const out = prev.slice();
      out[i] = next;
      return out;
    });
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const r = scoreAnswers(answers);
    writeBestIfBetter(r.correctCount, FLEET_VALUES.length);
    setBest(readBest());
    setSubmitted(true);
  }

  function handleReset() {
    setAnswers(["", "", "", "", ""]);
    setSubmitted(false);
  }

  if (!hydrated) return null;

  return (
    <div className="space-y-6">
      <div className="rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Fleet&apos;s 5 core values
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          Type each value in the fields below. Any order. Case and spacing
          don&apos;t matter.
        </p>
        {best ? (
          <p className="mt-2 text-xs text-zinc-500 dark:text-zinc-400">
            Best: {best.bestCorrect} / {best.bestTotal} correct.
          </p>
        ) : null}

        <form onSubmit={handleSubmit} className="mt-5 space-y-2">
          {answers.map((value, i) => {
            const perInput = result?.perInput[i];
            const isCounted = perInput?.counted ?? false;
            const isUnmatched = submitted && !perInput?.canonical;
            const isDuplicate =
              submitted && !!perInput?.canonical && !perInput?.counted;
            const borderClass = submitted
              ? isCounted
                ? "border-emerald-500 ring-1 ring-emerald-500"
                : isDuplicate
                  ? "border-amber-400 ring-1 ring-amber-400"
                  : "border-rose-400 ring-1 ring-rose-400"
              : "border-zinc-300 focus-within:border-zinc-500 dark:border-zinc-700 dark:focus-within:border-zinc-400";
            return (
              <div key={i} className="flex items-center gap-2">
                <span className="w-6 shrink-0 text-right font-mono text-xs text-zinc-500">
                  {i + 1}.
                </span>
                <input
                  type="text"
                  value={value}
                  onChange={(e) => updateAnswer(i, e.target.value)}
                  disabled={submitted}
                  placeholder={submitted ? "" : "Type a value..."}
                  autoComplete="off"
                  spellCheck={false}
                  className={`w-full rounded border bg-white px-2.5 py-1.5 text-sm text-zinc-900 placeholder:text-zinc-400 disabled:opacity-80 dark:bg-zinc-950 dark:text-zinc-100 dark:placeholder:text-zinc-500 ${borderClass}`}
                />
                {submitted ? (
                  <span className="w-4 shrink-0 text-center">
                    {isCounted ? (
                      <span
                        className="text-emerald-600 dark:text-emerald-400"
                        aria-label="correct"
                      >
                        ✓
                      </span>
                    ) : isDuplicate ? (
                      <span
                        className="text-amber-600 dark:text-amber-400"
                        aria-label="duplicate"
                        title="This is a Fleet value, but you already entered it in another field"
                      >
                        ↺
                      </span>
                    ) : isUnmatched ? (
                      <span
                        className="text-rose-600 dark:text-rose-400"
                        aria-label="not a Fleet value"
                      >
                        ✗
                      </span>
                    ) : null}
                  </span>
                ) : (
                  <span className="w-4 shrink-0" />
                )}
              </div>
            );
          })}

          <div className="pt-2">
            {submitted ? (
              <button
                type="button"
                onClick={handleReset}
                className="rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700"
              >
                Try again
              </button>
            ) : (
              <button
                type="submit"
                className="rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
                disabled={answers.every((a) => a.trim() === "")}
              >
                Submit
              </button>
            )}
          </div>
        </form>
      </div>

      {submitted ? (
        <div className="rounded border border-emerald-300 bg-emerald-50 p-5 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-100">
          <div className="font-semibold">
            {result!.correctCount} / {FLEET_VALUES.length} correct
          </div>
          <div className="mt-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
              The 5 values
            </div>
            <ul className="mt-2 space-y-1">
              {FLEET_VALUES.map((v) => (
                <li key={v.name} className="flex items-center gap-2">
                  <span aria-hidden>{v.emoji}</span>
                  <span className="font-medium">{v.name}</span>
                </li>
              ))}
            </ul>
          </div>
          <p className="mt-3 text-xs text-emerald-800 dark:text-emerald-200">
            Deep dive on each value:{" "}
            <a
              href="https://fleetdm.com/handbook/company#values"
              target="_blank"
              rel="noopener"
              className="underline"
            >
              fleetdm.com/handbook/company#values
            </a>
          </p>
          <p className="mt-3 text-xs text-emerald-800 dark:text-emerald-200">
            <span className="font-semibold">Fun fact:</span> Fleet&apos;s Slack
            has custom emojis for each value .{" "}
            <code className="rounded bg-emerald-100 px-1 font-mono text-[11px] text-emerald-900 dark:bg-emerald-900/60 dark:text-emerald-100">
              :empathy_value:
            </code>
            ,{" "}
            <code className="rounded bg-emerald-100 px-1 font-mono text-[11px] text-emerald-900 dark:bg-emerald-900/60 dark:text-emerald-100">
              :ownership_value:
            </code>
            ,{" "}
            <code className="rounded bg-emerald-100 px-1 font-mono text-[11px] text-emerald-900 dark:bg-emerald-900/60 dark:text-emerald-100">
              :results_value:
            </code>
            ,{" "}
            <code className="rounded bg-emerald-100 px-1 font-mono text-[11px] text-emerald-900 dark:bg-emerald-900/60 dark:text-emerald-100">
              :objectivity_value:
            </code>
            ,{" "}
            <code className="rounded bg-emerald-100 px-1 font-mono text-[11px] text-emerald-900 dark:bg-emerald-900/60 dark:text-emerald-100">
              :openness_value:
            </code>
            . Faster than typing the word.
          </p>
        </div>
      ) : null}
    </div>
  );
}
