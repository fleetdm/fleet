"use client";

import { useState } from "react";
import QuizFormatChip from "@/components/QuizFormatChip";
import { TIER_QUESTIONS, type TierQuestion } from "./quiz-questions";

const STORAGE_KEY = "fleet-atlas:tier-quiz-best";
const IN_PROGRESS_KEY = "fleet-atlas:tier-quiz-in-progress";

type BestRecord = {
  bestCorrect: number;
  total: number;
  bestTakenAt: number;
  lastCorrect?: number;
  lastTotal?: number;
  lastTakenAt?: number;
};

type InProgress = {
  orderIds: string[];
  index: number;
  correctCount: number;
};

function loadInProgress(): InProgress | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(IN_PROGRESS_KEY);
    return raw ? (JSON.parse(raw) as InProgress) : null;
  } catch {
    return null;
  }
}

function saveInProgress(data: InProgress | null) {
  if (typeof window === "undefined") return;
  try {
    if (data === null) localStorage.removeItem(IN_PROGRESS_KEY);
    else localStorage.setItem(IN_PROGRESS_KEY, JSON.stringify(data));
  } catch {
    /* quota */
  }
}

function loadBest(): BestRecord | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as BestRecord) : null;
  } catch {
    return null;
  }
}

function saveBest(record: BestRecord) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(record));
  } catch {
    /* quota */
  }
}

function clearBest() {
  if (typeof window === "undefined") return;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* ignore */
  }
}

function shuffle<T>(arr: T[]): T[] {
  const out = [...arr];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

function getInitialState() {
  const best = loadBest();
  const inProgress = loadInProgress();
  if (inProgress) {
    const byId = new Map(TIER_QUESTIONS.map((q) => [q.id, q]));
    const restored = inProgress.orderIds
      .map((id) => byId.get(id))
      .filter((q): q is TierQuestion => Boolean(q));
    if (restored.length === TIER_QUESTIONS.length) {
      return {
        order: restored,
        index: inProgress.index,
        correctCount: inProgress.correctCount,
        best,
      };
    }
  }
  return {
    order: typeof window === "undefined" ? [] : shuffle(TIER_QUESTIONS),
    index: 0,
    correctCount: 0,
    best,
  };
}

export default function TierQuiz() {
  const [initial] = useState(getInitialState);
  const [order, setOrder] = useState<TierQuestion[]>(initial.order);
  const [index, setIndex] = useState(initial.index);
  const [picked, setPicked] = useState<"free" | "premium" | null>(null);
  const [correctCount, setCorrectCount] = useState(initial.correctCount);
  const [best, setBest] = useState<BestRecord | null>(initial.best);
  const [finished, setFinished] = useState(false);

  const current = order[index];
  const total = order.length;

  const pick = (choice: "free" | "premium") => {
    if (picked) return;
    setPicked(choice);
    const nextCorrect =
      current.answer === choice ? correctCount + 1 : correctCount;
    if (current.answer === choice) {
      setCorrectCount(nextCorrect);
    }
    // Persist partial progress so a tab close doesn't lose it. Best score
    // only updates on full completion (see next()).
    saveInProgress({
      orderIds: order.map((q) => q.id),
      index,
      correctCount: nextCorrect,
    });
  };

  const next = () => {
    if (index + 1 >= total) {
      const finalCorrect = correctCount;
      const now = Date.now();
      const isNewBest = !best || finalCorrect > best.bestCorrect;
      const record: BestRecord = {
        bestCorrect: isNewBest ? finalCorrect : (best?.bestCorrect ?? finalCorrect),
        total: isNewBest ? total : (best?.total ?? total),
        bestTakenAt: isNewBest ? now : (best?.bestTakenAt ?? now),
        lastCorrect: finalCorrect,
        lastTotal: total,
        lastTakenAt: now,
      };
      saveBest(record);
      saveInProgress(null);
      setBest(record);
      setFinished(true);
      return;
    }
    const newIndex = index + 1;
    setIndex(newIndex);
    setPicked(null);
    saveInProgress({
      orderIds: order.map((q) => q.id),
      index: newIndex,
      correctCount,
    });
  };

  const restart = () => {
    setOrder(shuffle(TIER_QUESTIONS));
    setIndex(0);
    setPicked(null);
    setCorrectCount(0);
    setFinished(false);
    saveInProgress(null);
  };

  if (order.length === 0) return null;

  const progressPct = Math.round(((index + (picked ? 1 : 0)) / total) * 100);

  return (
    <section
      id="quiz"
      className="scroll-mt-6 rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20"
    >
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="binary" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Fleet Free or Fleet Premium?
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {TIER_QUESTIONS.length} features. Pick the tier each one needs. Answers
        reveal the specific gate.
      </p>

      {(best || loadInProgress()) && (
        <ScoreChips
          best={best}
          onReset={() => {
            if (confirm("Reset your best score and clear any quiz in progress?")) {
              clearBest();
              saveInProgress(null);
              setBest(null);
              setOrder(shuffle(TIER_QUESTIONS));
              setIndex(0);
              setPicked(null);
              setCorrectCount(0);
              setFinished(false);
            }
          }}
        />
      )}

      {finished ? (
        <FinishedCard
          correct={correctCount}
          total={total}
          onRestart={restart}
        />
      ) : null}
      {!finished && (
        <>
          <div className="mt-5 h-1.5 overflow-hidden rounded bg-zinc-200 dark:bg-zinc-800">
            <div
              className="h-full bg-emerald-500 transition-all"
              style={{ width: `${progressPct}%` }}
            />
          </div>
          <div className="mt-1 flex items-center justify-between text-[11px] font-mono text-zinc-500">
            <span>
              Score: {correctCount}
              {index > 0 || picked ? ` / ${index + (picked ? 1 : 0)}` : ""}
            </span>
            <span>
              {index + 1} / {total}
            </span>
          </div>

          <div className="mt-4 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
            <div className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
              This feature:
            </div>
            <div className="mt-1 text-base text-zinc-900 dark:text-zinc-100">
              {current.feature}
            </div>

            <div className="mt-5 grid grid-cols-2 gap-3">
              <TierButton
                label="Available on Fleet Free"
                hex="#5CABDF"
                selected={picked === "free"}
                correct={current.answer === "free"}
                revealed={picked !== null}
                onClick={() => pick("free")}
              />
              <TierButton
                label="Premium only"
                hex="#FAA669"
                selected={picked === "premium"}
                correct={current.answer === "premium"}
                revealed={picked !== null}
                onClick={() => pick("premium")}
              />
            </div>

            {/* Reserve vertical space so the card doesn't jump height when the
                explanation appears. 11rem accommodates ~3 lines of explanation
                + the Next button. */}
            <div className="mt-5 min-h-[11rem]">
              {picked && (
                <>
                  <div className="rounded border border-zinc-200 bg-zinc-50 p-3 text-sm text-zinc-800 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                      {picked === current.answer ? "Correct" : "Not quite"}
                    </div>
                    {current.explanation}
                  </div>

                  <button
                    onClick={next}
                    className="mt-4 w-full rounded border border-zinc-900 bg-zinc-900 px-3 py-2 text-sm font-semibold text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {index + 1 >= total ? "See results" : "Next question"}
                  </button>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function ScoreChips({
  best,
  onReset,
}: {
  best: BestRecord | null;
  onReset: () => void;
}) {
  return (
    <div className="mt-3 flex flex-wrap items-center gap-2">
      {best && (
        <>
          <span className="inline-flex items-center gap-2 rounded border border-emerald-300 bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-200">
            Best {best.bestCorrect}/{best.total}
          </span>
          {best.lastCorrect !== undefined &&
            best.lastTotal !== undefined &&
            best.lastCorrect !== best.bestCorrect && (
              <span className="inline-flex items-center gap-2 rounded bg-zinc-100 px-2 py-1 text-xs font-semibold text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                Last {best.lastCorrect}/{best.lastTotal}
              </span>
            )}
        </>
      )}
      <button
        onClick={onReset}
        className="rounded border border-zinc-300 bg-white px-2 py-1 text-xs font-semibold text-zinc-700 hover:border-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-500 dark:hover:bg-zinc-800"
      >
        Reset quiz
      </button>
    </div>
  );
}

function TierButton({
  label,
  hex,
  selected,
  correct,
  revealed,
  onClick,
}: {
  label: string;
  hex: string;
  selected: boolean;
  correct: boolean;
  revealed: boolean;
  onClick: () => void;
}) {
  let style =
    "border-zinc-300 bg-white text-zinc-800 hover:border-zinc-500 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200";
  if (revealed) {
    if (correct) {
      style =
        "border-emerald-500 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200";
    } else if (selected) {
      style =
        "border-rose-500 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200";
    } else {
      style =
        "border-zinc-200 bg-white text-zinc-400 opacity-60 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-500";
    }
  }
  return (
    <button
      onClick={onClick}
      disabled={revealed}
      className={`flex items-center justify-center gap-2 rounded border px-3 py-3 text-sm font-semibold transition-colors ${style}`}
    >
      <span
        className="h-2.5 w-2.5 shrink-0 rounded-full"
        style={{ backgroundColor: hex }}
      />
      {label}
    </button>
  );
}

function FinishedCard({
  correct,
  total,
  onRestart,
}: {
  correct: number;
  total: number;
  onRestart: () => void;
}) {
  const pct = Math.round((correct / total) * 100);
  const grade =
    pct >= 90
      ? { label: "Mastered", color: "text-emerald-700 dark:text-emerald-300" }
      : pct >= 70
        ? { label: "Solid", color: "text-sky-700 dark:text-sky-300" }
        : { label: "Keep practicing", color: "text-amber-700 dark:text-amber-300" };

  return (
    <div className="mt-5 rounded border border-zinc-200 bg-white p-6 text-center dark:border-zinc-800 dark:bg-zinc-950">
      <div className={`text-xs font-semibold uppercase tracking-widest ${grade.color}`}>
        {grade.label}
      </div>
      <div className="mt-2 text-4xl font-semibold text-zinc-900 dark:text-zinc-100">
        {correct} / {total}
      </div>
      <div className="mt-1 text-sm text-zinc-500">{pct}%</div>
      <button
        onClick={onRestart}
        className="mt-5 rounded border border-zinc-900 bg-zinc-900 px-3 py-1.5 text-sm font-semibold text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
      >
        Try again
      </button>
    </div>
  );
}


