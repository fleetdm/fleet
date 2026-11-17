"use client";

import { useMemo, useState } from "react";
import QuizFormatChip from "@/components/QuizFormatChip";
import { CRONS } from "./jobs";
import {
  CRON_QUESTIONS,
  pickAdaptive,
  type CronQuestion,
} from "./cron-questions";

const STORAGE_KEY = "fleet-atlas:cron-quiz-best";
const IN_PROGRESS_KEY = "fleet-atlas:cron-quiz-in-progress";

type BestRecord = {
  bestCorrect: number;
  bestTotal: number;
  bestFinalDifficulty: number;
  bestTakenAt: number;
};

type InProgress = {
  askedIds: string[];
  correctCount: number;
  difficulty: number;
  currentId: string | null;
};

type HistoryEntry = {
  question: CronQuestion;
  pickedIndex: number;
  correct: boolean;
};

function readBest(): BestRecord | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as BestRecord) : null;
  } catch {
    return null;
  }
}

function writeBest(r: BestRecord | null) {
  if (typeof window === "undefined") return;
  try {
    if (r === null) localStorage.removeItem(STORAGE_KEY);
    else localStorage.setItem(STORAGE_KEY, JSON.stringify(r));
  } catch {
    /* quota */
  }
}

function readInProgress(): InProgress | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(IN_PROGRESS_KEY);
    return raw ? (JSON.parse(raw) as InProgress) : null;
  } catch {
    return null;
  }
}

function writeInProgress(ip: InProgress | null) {
  if (typeof window === "undefined") return;
  try {
    if (ip === null) localStorage.removeItem(IN_PROGRESS_KEY);
    else localStorage.setItem(IN_PROGRESS_KEY, JSON.stringify(ip));
  } catch {
    /* quota */
  }
}

const DIFFICULTY_LABEL = (d: number) => {
  if (d <= 2) return "Easy";
  if (d <= 4) return "Introductory";
  if (d <= 6) return "Intermediate";
  if (d <= 8) return "Advanced";
  return "Expert";
};

const DIFFICULTY_COLOR = (d: number) => {
  if (d <= 2)
    return "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200";
  if (d <= 4)
    return "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200";
  if (d <= 6)
    return "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200";
  if (d <= 8)
    return "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200";
  return "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200";
};

function getInitialState() {
  const best = readBest();
  const ip = readInProgress();
  if (ip) {
    const curr = ip.currentId
      ? CRON_QUESTIONS.find((q) => q.id === ip.currentId)
      : null;
    if (curr) {
      const history = ip.askedIds
        .filter((id) => id !== ip.currentId)
        .map((id) => {
          const q = CRON_QUESTIONS.find((x) => x.id === id)!;
          const wasCorrect =
            ip.correctCount >
            ip.askedIds.filter(
              (xId) =>
                xId !== id &&
                CRON_QUESTIONS.findIndex((x) => x.id === xId) <
                  CRON_QUESTIONS.findIndex((x) => x.id === id),
            ).length;
          return { question: q, pickedIndex: -1, correct: wasCorrect };
        });
      return {
        current: curr,
        difficulty: ip.difficulty,
        history,
        best,
      };
    }
  }
  return {
    current: typeof window === "undefined" ? null : pickAdaptive(4, new Set()),
    difficulty: 4,
    history: [] as HistoryEntry[],
    best,
  };
}

export default function CronQuiz() {
  const [initial] = useState(getInitialState);
  const [current, setCurrent] = useState<CronQuestion | null>(initial.current);
  const [difficulty, setDifficulty] = useState(initial.difficulty);
  const [picked, setPicked] = useState<number | null>(null);
  const [history, setHistory] = useState<HistoryEntry[]>(initial.history);
  const [best, setBest] = useState<BestRecord | null>(initial.best);
  const [showDetail, setShowDetail] = useState(false);

  const askedIds = useMemo(
    () =>
      new Set([
        ...history.map((h) => h.question.id),
        ...(current ? [current.id] : []),
      ]),
    [history, current],
  );
  const correctCount = history.filter((h) => h.correct).length;
  const askedCount = history.length;
  const answered = picked !== null;
  const isCorrect = answered && current && picked === current.correctIndex;
  const relatedCron = current?.cronName
    ? CRONS.find((c) => c.name === current.cronName)
    : null;

  const submit = () => {
    if (!current || picked === null) return;
    const correct = picked === current.correctIndex;
    const entry: HistoryEntry = {
      question: current,
      pickedIndex: picked,
      correct,
    };
    const newHistory = [...history, entry];
    setHistory(newHistory);

    // Running best: compare correct count against stored best.
    const newCorrect = newHistory.filter((h) => h.correct).length;
    const newTotal = newHistory.length;
    const curBest = readBest();
    const isBetter =
      !curBest ||
      newCorrect > curBest.bestCorrect ||
      (newCorrect === curBest.bestCorrect && newTotal > curBest.bestTotal);
    if (isBetter) {
      const rec: BestRecord = {
        bestCorrect: newCorrect,
        bestTotal: newTotal,
        bestFinalDifficulty: difficulty,
        bestTakenAt: Date.now(),
      };
      writeBest(rec);
      setBest(rec);
    }
  };

  const next = () => {
    if (!current || picked === null) return;
    const correct = picked === current.correctIndex;
    const nextDiff = Math.max(1, Math.min(10, difficulty + (correct ? 1 : -1)));
    setDifficulty(nextDiff);
    const q = pickAdaptive(nextDiff, askedIds);
    setPicked(null);
    setShowDetail(false);
    setCurrent(q);
    writeInProgress({
      askedIds: Array.from(askedIds).concat(q ? [q.id] : []),
      correctCount: history.filter((h) => h.correct).length + (correct ? 1 : 0),
      difficulty: nextDiff,
      currentId: q?.id ?? null,
    });
  };

  const restart = () => {
    writeInProgress(null);
    setHistory([]);
    setPicked(null);
    setShowDetail(false);
    setDifficulty(4);
    const q = pickAdaptive(4, new Set());
    setCurrent(q);
  };

  const resetBest = () => {
    if (!confirm("Reset your best score for this quiz?")) return;
    writeBest(null);
    setBest(null);
  };

  if (!current) {
    return (
      <section className="scroll-mt-6 rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          You&rsquo;ve answered every question.
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          Final: {correctCount}/{askedCount} correct at ending difficulty{" "}
          {DIFFICULTY_LABEL(difficulty)}.
        </p>
        <button
          onClick={restart}
          className="mt-4 rounded border border-zinc-900 bg-zinc-900 px-3 py-1.5 text-sm font-semibold text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900"
        >
          Try again
        </button>
      </section>
    );
  }

  return (
    <section
      id="quiz"
      className="scroll-mt-6 rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20"
    >
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="adaptive-multiple-choice" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Which cron? Adaptive quiz
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Each question picks the right cron for a symptom or behavior. Correct
        answers raise the difficulty; wrong answers lower it.
      </p>

      {best && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-2 rounded border border-emerald-300 bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-200">
            Best {best.bestCorrect}/{best.bestTotal} at{" "}
            {DIFFICULTY_LABEL(best.bestFinalDifficulty)}
          </span>
          <button
            onClick={resetBest}
            className="rounded border border-zinc-300 bg-white px-2 py-1 text-xs font-semibold text-zinc-700 hover:border-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          >
            Reset
          </button>
        </div>
      )}

      <div className="mt-5 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="mb-3 flex items-center justify-between gap-2 text-xs">
          <span
            className={`rounded px-2 py-0.5 font-semibold ${DIFFICULTY_COLOR(current.difficulty)}`}
          >
            {DIFFICULTY_LABEL(current.difficulty)} · {current.difficulty}/10
          </span>
          <span className="font-mono text-zinc-500">
            {correctCount}/{askedCount} correct · {CRON_QUESTIONS.length - askedCount}{" "}
            remaining
          </span>
        </div>

        <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
          {current.question}
        </h3>

        <ul className="mt-4 space-y-2">
          {current.options.map((opt, i) => {
            const isPicked = picked === i;
            const isAnswer = i === current.correctIndex;
            let style =
              "border-zinc-200 hover:border-zinc-400 dark:border-zinc-700 dark:hover:border-zinc-500";
            if (answered) {
              if (isAnswer)
                style = "border-emerald-500 bg-emerald-50 dark:bg-emerald-950/40";
              else if (isPicked && !isAnswer)
                style = "border-rose-500 bg-rose-50 dark:bg-rose-950/40";
              else style = "border-zinc-200 opacity-60 dark:border-zinc-800";
            } else if (isPicked) {
              style =
                "border-zinc-900 bg-zinc-50 dark:border-zinc-100 dark:bg-zinc-800";
            }
            return (
              <li key={i}>
                <button
                  disabled={answered}
                  onClick={() => setPicked(i)}
                  className={`flex w-full items-start gap-3 rounded border p-3 text-left transition-colors ${style}`}
                >
                  <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-zinc-400 font-mono text-[11px] text-zinc-600 dark:border-zinc-500 dark:text-zinc-300">
                    {String.fromCharCode(65 + i)}
                  </span>
                  <span className="text-sm text-zinc-900 dark:text-zinc-100">
                    {opt}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>

        {answered && (
          <div
            className={`mt-4 rounded border p-3 text-sm ${
              isCorrect
                ? "border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-100"
                : "border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-100"
            }`}
          >
            <div className="font-semibold">
              {isCorrect ? "Correct." : "Not quite."}
            </div>
            <p className="mt-1 text-xs">{current.explanation}</p>
            {relatedCron && (
              <button
                onClick={() => setShowDetail((v) => !v)}
                className="mt-2 inline-block text-xs underline"
              >
                {showDetail ? "Hide" : "Show"} the {relatedCron.name} entry{" "}
                {showDetail ? "↑" : "↓"}
              </button>
            )}
          </div>
        )}

        {relatedCron && showDetail && (
          <div className="mt-3 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
            <div className="flex items-baseline justify-between gap-3">
              <code className="font-mono text-sm text-zinc-900 dark:text-zinc-100">
                {relatedCron.name}
              </code>
              <span className="text-xs text-zinc-500">
                {relatedCron.intervalMinutes < 1
                  ? `${Math.round(relatedCron.intervalMinutes * 60)}s`
                  : relatedCron.intervalMinutes < 60
                    ? `${relatedCron.intervalMinutes}m`
                    : relatedCron.intervalMinutes < 60 * 24
                      ? `${relatedCron.intervalMinutes / 60}h`
                      : `${relatedCron.intervalMinutes / (60 * 24)}d`}
              </span>
            </div>
            <p className="mt-2 text-sm text-zinc-800 dark:text-zinc-200">
              {relatedCron.purpose}
            </p>
            <p className="mt-2 text-xs text-zinc-500">
              Source: <code>{relatedCron.file}</code> · Topic: {relatedCron.topic}
            </p>
          </div>
        )}

        <div className="mt-5 flex items-center justify-between">
          <div className="text-xs text-zinc-500">
            {answered
              ? isCorrect
                ? "Next question will be harder."
                : "Next question will be easier."
              : picked === null
                ? "Pick an answer."
                : "Submit when ready."}
          </div>
          <div className="flex items-center gap-2">
            {!answered && picked !== null && (
              <button
                onClick={submit}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Submit
              </button>
            )}
            {answered && (
              <button
                onClick={next}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Next question →
              </button>
            )}
            <button
              onClick={restart}
              className="rounded border border-zinc-300 bg-white px-3 py-2 text-xs text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
            >
              Restart
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
