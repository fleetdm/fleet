"use client";

import { useMemo, useState } from "react";
import { useIsHydrated } from "@/lib/use-is-hydrated";
import { REVIEW_AI_QUESTIONS, type ReviewAIQuestion } from "./questions";

const IN_PROGRESS_KEY = "fleet-atlas:review-ai:in-progress";
const STORAGE_KEY = "fleet-atlas:review-ai:best";

type Attempt = {
  questionId: string;
  firstTryCorrect: boolean;
  wrongClicks: number;
};

type Progress = {
  order: string[];
  index: number;
  attempts: Attempt[];
};

type Best = {
  bestCorrect: number;
  bestTotal: number;
  bestAttempts: number;
  takenAt?: number;
};

function shuffle<T>(arr: T[]): T[] {
  const out = arr.slice();
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

function readProgress(): Progress | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(IN_PROGRESS_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Progress;
  } catch {
    return null;
  }
}

function writeProgress(p: Progress | null) {
  if (typeof window === "undefined") return;
  if (p === null) {
    localStorage.removeItem(IN_PROGRESS_KEY);
    return;
  }
  localStorage.setItem(IN_PROGRESS_KEY, JSON.stringify(p));
}

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

function writeBestIfBetter(attempts: Attempt[]) {
  if (typeof window === "undefined") return;
  const correct = attempts.filter((a) => a.firstTryCorrect).length;
  const total = attempts.length;
  const current = readBest();
  const prevPct = current ? current.bestCorrect / current.bestTotal : -1;
  const nextPct = total === 0 ? 0 : correct / total;
  if (!current || nextPct > prevPct) {
    const next: Best = {
      bestCorrect: correct,
      bestTotal: total,
      bestAttempts: total,
      takenAt: Date.now(),
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  }
}

type ClickState =
  | { kind: "unanswered" }
  | { kind: "wrong"; clickedLine: number }
  | { kind: "revealed"; firstTryCorrect: boolean; wrongClicks: number };

export default function ReviewAIQuiz() {
  const hydrated = useIsHydrated();
  const [progress, setProgress] = useState<Progress | null>(() => {
    const existing = readProgress();
    return existing && existing.index < existing.order.length ? existing : null;
  });
  const [click, setClick] = useState<ClickState>({ kind: "unanswered" });
  const [wrongLines, setWrongLines] = useState<Set<number>>(new Set());

  function startNewRound() {
    const order = shuffle(REVIEW_AI_QUESTIONS.map((q) => q.id));
    const next: Progress = { order, index: 0, attempts: [] };
    setProgress(next);
    writeProgress(next);
    setClick({ kind: "unanswered" });
    setWrongLines(new Set());
  }

  const current: ReviewAIQuestion | null = useMemo(() => {
    if (!progress) return null;
    const id = progress.order[progress.index];
    return REVIEW_AI_QUESTIONS.find((q) => q.id === id) ?? null;
  }, [progress]);

  function handleLineClick(lineNumber: number) {
    if (!current) return;
    if (click.kind === "revealed") return;
    const flagged = current.flaggedLines.includes(lineNumber);
    if (flagged) {
      const wrongCount = wrongLines.size;
      setClick({
        kind: "revealed",
        firstTryCorrect: wrongCount === 0,
        wrongClicks: wrongCount,
      });
    } else {
      setWrongLines((prev) => {
        const n = new Set(prev);
        n.add(lineNumber);
        return n;
      });
      setClick({ kind: "wrong", clickedLine: lineNumber });
    }
  }

  function handleNext() {
    if (!progress || !current || click.kind !== "revealed") return;
    const attempt: Attempt = {
      questionId: current.id,
      firstTryCorrect: click.firstTryCorrect,
      wrongClicks: click.wrongClicks,
    };
    const nextAttempts = [...progress.attempts, attempt];
    const nextIndex = progress.index + 1;
    if (nextIndex >= progress.order.length) {
      writeBestIfBetter(nextAttempts);
      writeProgress(null);
      setProgress({ ...progress, index: nextIndex, attempts: nextAttempts });
    } else {
      const next: Progress = {
        ...progress,
        index: nextIndex,
        attempts: nextAttempts,
      };
      setProgress(next);
      writeProgress(next);
    }
    setClick({ kind: "unanswered" });
    setWrongLines(new Set());
  }

  function handleGiveUp() {
    if (!current || click.kind === "revealed") return;
    setClick({
      kind: "revealed",
      firstTryCorrect: false,
      wrongClicks: wrongLines.size,
    });
  }

  if (!hydrated) return null;

  if (!progress) {
    const best = readBest();
    return (
      <div className="rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Review the AI
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          AI drafts a Fleet-shaped artifact. Click the line that first signals
          a mistake. Each question teaches a Fleet convention AI commonly
          fumbles.
        </p>
        {best ? (
          <p className="mt-3 text-xs text-zinc-500 dark:text-zinc-400">
            Best: {best.bestCorrect} / {best.bestTotal} first-try correct.
          </p>
        ) : null}
        <button
          onClick={startNewRound}
          className="mt-4 rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700"
        >
          Start a round
        </button>
      </div>
    );
  }

  const done = progress.index >= progress.order.length;

  if (done) {
    const correct = progress.attempts.filter((a) => a.firstTryCorrect).length;
    const total = progress.attempts.length;
    return (
      <div className="rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Round complete
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          {correct} / {total} first-try correct.
        </p>
        <ul className="mt-3 space-y-1 text-xs text-zinc-600 dark:text-zinc-400">
          {progress.attempts.map((a) => {
            const q = REVIEW_AI_QUESTIONS.find((x) => x.id === a.questionId);
            return (
              <li key={a.questionId}>
                <span
                  className={
                    a.firstTryCorrect
                      ? "text-emerald-700 dark:text-emerald-300"
                      : "text-amber-700 dark:text-amber-300"
                  }
                >
                  {a.firstTryCorrect ? "✓" : "·"}
                </span>{" "}
                {q?.title}
                {a.wrongClicks > 0 ? ` (${a.wrongClicks} wrong click${a.wrongClicks === 1 ? "" : "s"})` : ""}
              </li>
            );
          })}
        </ul>
        <button
          onClick={startNewRound}
          className="mt-4 rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700"
        >
          Start another round
        </button>
      </div>
    );
  }

  if (!current) return null;

  const lines = current.code.split("\n");
  const revealed = click.kind === "revealed";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <div className="text-xs uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
          {progress.index + 1} / {progress.order.length} · {current.area}
        </div>
        <div className="text-xs text-zinc-500 dark:text-zinc-400">
          Click the line that first signals a coding convention mistake.
        </div>
      </div>

      <div className="rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
          {current.title}
        </h3>
        <p className="mt-1 text-sm text-zinc-700 dark:text-zinc-300">
          {current.context}
        </p>
      </div>

      <div className="overflow-hidden rounded border border-zinc-200 bg-zinc-900 dark:border-zinc-800">
        <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-3 py-1.5">
          <span className="font-mono text-[11px] uppercase tracking-wide text-zinc-400">
            {current.language}
          </span>
          <button
            onClick={handleGiveUp}
            disabled={revealed}
            className="rounded border border-zinc-700 bg-zinc-900 px-2 py-0.5 text-[11px] text-zinc-300 hover:bg-zinc-800 disabled:opacity-40"
          >
            Reveal
          </button>
        </div>
        <pre className="m-0 overflow-x-auto p-0 font-mono text-[12.5px] leading-5 text-zinc-100">
          {lines.map((line, i) => {
            const lineNumber = i + 1;
            const isFlagged = current.flaggedLines.includes(lineNumber);
            const wasWrong = wrongLines.has(lineNumber);
            const highlight = (() => {
              if (revealed && isFlagged)
                return "bg-emerald-900/60 border-l-2 border-emerald-500";
              if (wasWrong)
                return "bg-rose-900/40 border-l-2 border-rose-500";
              return "border-l-2 border-transparent";
            })();
            const disabled = revealed;
            return (
              <button
                key={i}
                onClick={() => handleLineClick(lineNumber)}
                disabled={disabled}
                className={`flex w-full gap-3 px-3 text-left font-mono text-[12.5px] leading-5 ${highlight} ${
                  disabled ? "cursor-default" : "cursor-pointer hover:bg-zinc-800"
                }`}
              >
                <span className="w-6 shrink-0 select-none text-right text-zinc-600">
                  {lineNumber}
                </span>
                <span className="whitespace-pre text-zinc-100">
                  {line || " "}
                </span>
              </button>
            );
          })}
        </pre>
      </div>

      {click.kind === "wrong" ? (
        <div className="rounded border border-rose-300 bg-rose-50 p-3 text-sm text-rose-900 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-200">
          That line is fine. Keep looking.
        </div>
      ) : null}

      {revealed ? (
        <div className="space-y-3 rounded border border-emerald-300 bg-emerald-50 p-4 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-100">
          <div>
            <div className="font-semibold">
              {click.kind === "revealed" && click.firstTryCorrect
                ? "First try."
                : click.kind === "revealed" && click.wrongClicks === 0
                  ? "Revealed."
                  : "Got it."}
            </div>
            <p className="mt-1">{current.mistake}</p>
          </div>
          <div>
            <div className="text-xs font-semibold uppercase tracking-wide">
              Fix
            </div>
            <p className="mt-1">{current.fix}</p>
          </div>
          {current.conventionLink ? (
            <div className="text-xs">
              <span className="font-semibold uppercase tracking-wide">
                Convention:{" "}
              </span>
              <a
                href={current.conventionLink.href}
                target="_blank"
                rel="noopener"
                className="underline"
              >
                {current.conventionLink.label}
              </a>
            </div>
          ) : null}
          <button
            onClick={handleNext}
            className="rounded border border-emerald-600 bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700"
          >
            {progress.index + 1 >= progress.order.length
              ? "See results"
              : "Next question"}
          </button>
        </div>
      ) : null}
    </div>
  );
}
