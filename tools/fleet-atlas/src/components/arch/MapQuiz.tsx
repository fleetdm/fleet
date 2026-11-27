"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import {
  MAP_BUCKETS,
  pickAdaptiveMap,
  poolForMapBucket,
  type MapBucket,
  type MapQuestion,
} from "./map-questions";
import { getNode, GROUP_META } from "./graph-data";
import QuizFormatChip from "@/components/QuizFormatChip";

type HistoryEntry = {
  question: MapQuestion;
  pickedIndex: number;
  correct: boolean;
};

type ResultEntry = {
  bucketLabel: string;
  bucketSlug: string;
  correct: number;
  total: number;
  finalDifficulty: number;
  takenAt: number;
  bestCorrect: number;
  bestTotal: number;
  bestFinalDifficulty: number;
  bestTakenAt: number;
};

type LastResult = ResultEntry;

const STORAGE_KEY = "fleet-atlas:quiz-results:architecture";
const IN_PROGRESS_KEY = "fleet-atlas:quiz-in-progress:architecture";

type InProgress = {
  bucketSlug: string;
  difficulty: number;
  history: HistoryEntry[];
  current: MapQuestion | null;
  picked: number | null;
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
    /* quota or private mode */
  }
}

function loadResults(): Record<string, ResultEntry> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, Partial<ResultEntry>>;
    const out: Record<string, ResultEntry> = {};
    for (const [k, v] of Object.entries(parsed)) {
      if (v.bucketSlug == null) continue;
      out[k] = {
        bucketLabel: v.bucketLabel ?? k,
        bucketSlug: v.bucketSlug,
        correct: v.correct ?? 0,
        total: v.total ?? 0,
        finalDifficulty: v.finalDifficulty ?? 5,
        takenAt: v.takenAt ?? 0,
        bestCorrect: v.bestCorrect ?? v.correct ?? 0,
        bestTotal: v.bestTotal ?? v.total ?? 0,
        bestFinalDifficulty: v.bestFinalDifficulty ?? v.finalDifficulty ?? 5,
        bestTakenAt: v.bestTakenAt ?? v.takenAt ?? 0,
      };
    }
    return out;
  } catch {
    return {};
  }
}

function saveResults(map: Record<string, ResultEntry>) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch {
    /* quota or private mode */
  }
}

function pctOf(correct: number, total: number): number {
  return total > 0 ? correct / total : 0;
}

const DIFFICULTY_LABEL = (d: number) => {
  if (d <= 2) return "Easy";
  if (d <= 4) return "Introductory";
  if (d <= 6) return "Intermediate";
  if (d <= 8) return "Advanced";
  return "Expert";
};

const DIFFICULTY_COLOR = (d: number): string => {
  if (d <= 2)
    return "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200";
  if (d <= 4) return "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200";
  if (d <= 6)
    return "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200";
  if (d <= 8)
    return "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200";
  return "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200";
};

const BUCKET_COLOR: Record<string, string> = {
  system:
    "border-zinc-300 bg-zinc-50 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-zinc-600",
  software:
    "border-emerald-300 bg-emerald-50 hover:border-emerald-400 dark:border-emerald-800 dark:bg-emerald-950/40 dark:hover:border-emerald-700",
  mdm:
    "border-rose-300 bg-rose-50 hover:border-rose-400 dark:border-rose-800 dark:bg-rose-950/40 dark:hover:border-rose-700",
  policies:
    "border-indigo-300 bg-indigo-50 hover:border-indigo-400 dark:border-indigo-800 dark:bg-indigo-950/40 dark:hover:border-indigo-700",
  reports:
    "border-amber-300 bg-amber-50 hover:border-amber-400 dark:border-amber-800 dark:bg-amber-950/40 dark:hover:border-amber-700",
};

function computeInitialState(target: string | null) {
  const empty = {
    bucket: null as MapBucket | null,
    current: null as MapQuestion | null,
    difficulty: 5,
    picked: null as number | null,
    history: [] as HistoryEntry[],
  };
  if (typeof window === "undefined") return empty;
  const stored = loadInProgress();
  if (target) {
    const b = MAP_BUCKETS.find((bb) => bb.slug === target);
    if (!b) return empty;
    if (stored && stored.bucketSlug === target && stored.current) {
      return {
        bucket: b,
        current: stored.current,
        difficulty: stored.difficulty,
        picked: stored.picked,
        history: stored.history,
      };
    }
    return { ...empty, bucket: b, difficulty: b.startDifficulty ?? 5 };
  }
  if (stored && stored.current) {
    const b = MAP_BUCKETS.find((bb) => bb.slug === stored.bucketSlug);
    if (b) {
      return {
        bucket: b,
        current: stored.current,
        difficulty: stored.difficulty,
        picked: stored.picked,
        history: stored.history,
      };
    }
  }
  return empty;
}

export default function MapQuiz() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [initial] = useState(() =>
    computeInitialState(searchParams.get("bucket")),
  );
  const [bucket, setBucket] = useState<MapBucket | null>(initial.bucket);
  const [current, setCurrent] = useState<MapQuestion | null>(initial.current);
  const [difficulty, setDifficulty] = useState(initial.difficulty);
  const [picked, setPicked] = useState<number | null>(initial.picked);
  const [history, setHistory] = useState<HistoryEntry[]>(initial.history);
  const [results, setResults] = useState<Record<string, LastResult>>(() =>
    loadResults(),
  );
  const [justFinished, setJustFinished] = useState<string | null>(null);
  const [showNodeCard, setShowNodeCard] = useState(false);

  // Persist in-progress state on any change
  useEffect(() => {
    if (!bucket || !current) {
      if (!bucket) saveInProgress(null);
      return;
    }
    saveInProgress({
      bucketSlug: bucket.slug,
      difficulty,
      history,
      current,
      picked,
    });
  }, [bucket, difficulty, history, current, picked]);

  const pool = useMemo(() => (bucket ? poolForMapBucket(bucket) : []), [bucket]);

  const askedKeys = useMemo(() => {
    const s = new Set<string>();
    for (const h of history)
      s.add(`${h.question.nodeSlug}:${h.question.question}`);
    if (current) s.add(`${current.nodeSlug}:${current.question}`);
    return s;
  }, [history, current]);

  const correctCount = history.filter((h) => h.correct).length;
  const total = history.length;

  const start = (b: MapBucket) => {
    const d = b.startDifficulty ?? 5;
    const q = pickAdaptiveMap(d, new Set(), poolForMapBucket(b));
    setBucket(b);
    setHistory([]);
    setPicked(null);
    setDifficulty(d);
    setCurrent(q ?? null);
    setJustFinished(null);
  };

  const exitToPicker = () => {
    if (bucket && history.length > 0) {
      const correct = history.filter((h) => h.correct).length;
      const total = history.length;
      const prior = results[bucket.slug];
      const priorBest = pctOf(prior?.bestCorrect ?? 0, prior?.bestTotal ?? 0);
      const thisPct = pctOf(correct, total);
      const priorCorrectCount = prior?.bestCorrect ?? 0;
      const isBetter =
        correct > priorCorrectCount ||
        (correct === priorCorrectCount && thisPct > priorBest);
      const entry: ResultEntry = {
        bucketLabel: bucket.label,
        bucketSlug: bucket.slug,
        correct,
        total,
        finalDifficulty: difficulty,
        takenAt: Date.now(),
        bestCorrect: isBetter ? correct : prior?.bestCorrect ?? correct,
        bestTotal: isBetter ? total : prior?.bestTotal ?? total,
        bestFinalDifficulty: isBetter
          ? difficulty
          : prior?.bestFinalDifficulty ?? difficulty,
        bestTakenAt: isBetter ? Date.now() : prior?.bestTakenAt ?? Date.now(),
      };
      const next = { ...results, [bucket.slug]: entry };
      setResults(next);
      saveResults(next);
      setJustFinished(bucket.slug);
    }
    setBucket(null);
    setCurrent(null);
    setHistory([]);
    setPicked(null);
    setDifficulty(5);
    router.push("/self-assessments");
  };

  const clearResult = (slug: string) => {
    const next = { ...results };
    delete next[slug];
    setResults(next);
    saveResults(next);
    if (justFinished === slug) setJustFinished(null);
  };

  const persistProgress = (
    nextHistory: HistoryEntry[],
    curDifficulty: number,
  ) => {
    if (!bucket || nextHistory.length === 0) return;
    const correct = nextHistory.filter((h) => h.correct).length;
    const total = nextHistory.length;
    setResults((prev) => {
      const prior = prev[bucket.slug];
      const priorBest = pctOf(prior?.bestCorrect ?? 0, prior?.bestTotal ?? 0);
      const thisPct = pctOf(correct, total);
      const priorCorrectCount = prior?.bestCorrect ?? 0;
      const isBetter =
        correct > priorCorrectCount ||
        (correct === priorCorrectCount && thisPct > priorBest);
      const entry: ResultEntry = {
        bucketLabel: bucket.label,
        bucketSlug: bucket.slug,
        correct,
        total,
        finalDifficulty: curDifficulty,
        takenAt: Date.now(),
        bestCorrect: isBetter ? correct : prior?.bestCorrect ?? correct,
        bestTotal: isBetter ? total : prior?.bestTotal ?? total,
        bestFinalDifficulty: isBetter
          ? curDifficulty
          : prior?.bestFinalDifficulty ?? curDifficulty,
        bestTakenAt: isBetter ? Date.now() : prior?.bestTakenAt ?? Date.now(),
      };
      const next = { ...prev, [bucket.slug]: entry };
      saveResults(next);
      return next;
    });
  };

  const submit = () => {
    if (picked == null || !current) return;
    const correct = picked === current.correctIndex;
    const newEntry: HistoryEntry = {
      question: current,
      pickedIndex: picked,
      correct,
    };
    const nextHistory = [...history, newEntry];
    setHistory(nextHistory);
    persistProgress(nextHistory, difficulty);
  };

  const next = () => {
    if (!current || picked == null) return;
    const correct = picked === current.correctIndex;
    const ceiling = bucket?.maxDifficulty ?? 10;
    const nextDifficulty = Math.max(
      1,
      Math.min(ceiling, difficulty + (correct ? 1 : -1)),
    );
    setDifficulty(nextDifficulty);
    const asked = new Set(askedKeys);
    const q = pickAdaptiveMap(nextDifficulty, asked, pool);
    setPicked(null);
    setShowNodeCard(false);
    setCurrent(q);
  };

  const reset = () => {
    if (!bucket) return;
    const d = bucket.startDifficulty ?? 5;
    setHistory([]);
    setPicked(null);
    setDifficulty(d);
    setCurrent(pickAdaptiveMap(d, new Set(), pool));
  };

  const answered =
    picked != null &&
    history.length > 0 &&
    history[history.length - 1].question === current;
  const latest = answered ? history[history.length - 1] : null;
  const nodeForLatest = latest ? getNode(latest.question.nodeSlug) : null;

  // Picker
  if (!bucket) {
    const justFinishedResult = justFinished ? results[justFinished] : null;
    const pct = justFinishedResult
      ? Math.round((justFinishedResult.correct / Math.max(1, justFinishedResult.total)) * 100)
      : 0;
    return (
      <div className="mx-auto max-w-4xl px-10 py-10">
        {justFinishedResult ? (
          <div
            className={`mb-5 rounded border p-4 ${BUCKET_COLOR[justFinishedResult.bucketSlug] ?? "border-zinc-300 bg-white dark:border-zinc-700 dark:bg-zinc-900"}`}
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-[11px] font-semibold uppercase tracking-wide text-zinc-500">
                  Just finished
                </div>
                <div className="mt-1 text-base font-semibold text-zinc-900 dark:text-zinc-100">
                  {justFinishedResult.bucketLabel}
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-zinc-700 dark:text-zinc-300">
                  <span>
                    <span className="font-semibold text-zinc-900 dark:text-zinc-100">
                      {justFinishedResult.correct}/{justFinishedResult.total}
                    </span>{" "}
                    <span className="text-zinc-500">({pct}%)</span>
                  </span>
                  <span className="text-zinc-400">·</span>
                  <span>
                    Ended at difficulty{" "}
                    <span className="font-semibold text-zinc-900 dark:text-zinc-100">
                      {justFinishedResult.finalDifficulty}/10
                    </span>{" "}
                    <span
                      className={`ml-1 rounded px-1.5 py-0.5 text-[10px] font-semibold ${DIFFICULTY_COLOR(justFinishedResult.finalDifficulty)}`}
                    >
                      {DIFFICULTY_LABEL(justFinishedResult.finalDifficulty)}
                    </span>
                  </span>
                </div>
              </div>
              <button
                onClick={() => setJustFinished(null)}
                className="text-xs text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-200"
                title="Dismiss"
              >
                ✕
              </button>
            </div>
          </div>
        ) : null}

        <div className="mb-5">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            Pick a map to be quizzed on
          </h2>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Questions range from Easy (where a node sits) to Expert (why it&apos;s
            built this way). Scores persist across visits.
          </p>
        </div>
        <ul className="grid gap-3 md:grid-cols-2">
          {MAP_BUCKETS.map((b) => {
            const count = poolForMapBucket(b).length;
            const prior = results[b.slug];
            const priorPct = prior
              ? Math.round((prior.correct / Math.max(1, prior.total)) * 100)
              : 0;
            return (
              <li key={b.slug}>
                <button
                  onClick={() => start(b)}
                  className={`w-full rounded border p-4 text-left transition-colors ${BUCKET_COLOR[b.slug] ?? ""}`}
                >
                  <div className="flex items-baseline justify-between gap-3">
                    <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                      {b.label}
                    </h3>
                    <span className="text-xs text-zinc-500">
                      {count} questions
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-zinc-700 dark:text-zinc-300">
                    {b.description}
                  </p>
                  {prior ? (
                    <div className="mt-3 space-y-1 border-t border-zinc-300/60 pt-2 text-xs dark:border-zinc-700/60">
                      <div className="flex items-center gap-2">
                        <span className="w-10 font-semibold uppercase tracking-wide text-zinc-500">
                          Last
                        </span>
                        <span className="text-zinc-700 dark:text-zinc-300">
                          {prior.correct}/{prior.total} ({priorPct}%)
                        </span>
                        <span
                          className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${DIFFICULTY_COLOR(prior.finalDifficulty)}`}
                        >
                          {prior.finalDifficulty}/10
                        </span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            clearResult(b.slug);
                          }}
                          className="ml-auto text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
                          title="Clear last + best"
                        >
                          ✕
                        </button>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="w-10 font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-400">
                          Best
                        </span>
                        <span className="text-zinc-700 dark:text-zinc-300">
                          {prior.bestCorrect}/{prior.bestTotal} (
                          {Math.round(pctOf(prior.bestCorrect, prior.bestTotal) * 100)}%)
                        </span>
                        <span
                          className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${DIFFICULTY_COLOR(prior.bestFinalDifficulty)}`}
                        >
                          {prior.bestFinalDifficulty}/10
                        </span>
                      </div>
                    </div>
                  ) : null}
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    );
  }

  // Exhausted
  if (!current) {
    return (
      <div className="mx-auto max-w-3xl px-10 py-10">
        <div className="rounded border border-zinc-200 bg-white p-8 text-center dark:border-zinc-800 dark:bg-zinc-950">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            You&apos;ve exhausted the {bucket.label} pool
          </h2>
          <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
            Final score: {correctCount}/{total} (
            {Math.round((correctCount / Math.max(1, total)) * 100)}%). Final
            difficulty reached: {difficulty}/10 ({DIFFICULTY_LABEL(difficulty)}).
          </p>
          <div className="mt-5 flex items-center justify-center gap-2">
            <button
              onClick={reset}
              className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
            >
              Restart {bucket.label}
            </button>
            <button
              onClick={exitToPicker}
              className="rounded border border-zinc-300 bg-white px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
            >
              Pick another map
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-10 py-10">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded border border-zinc-200 bg-white p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950">
        <div className="flex items-center gap-4">
          <div>
            <div className="text-[11px] uppercase tracking-wide text-zinc-500">
              Assessment
            </div>
            <div className="flex items-center gap-2">
              <div className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                {bucket.label}
              </div>
              <QuizFormatChip format="adaptive-multiple-choice" />
            </div>
          </div>
          <div>
            <div className="text-[11px] uppercase tracking-wide text-zinc-500">
              Score
            </div>
            <div className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
              {correctCount}/{total}
            </div>
          </div>
          <div>
            <div className="text-[11px] uppercase tracking-wide text-zinc-500">
              Difficulty
            </div>
            <div className="flex items-center gap-2">
              <span className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                {difficulty}/10
              </span>
              <span
                className={`rounded px-2 py-0.5 text-[11px] font-semibold ${DIFFICULTY_COLOR(difficulty)}`}
              >
                {DIFFICULTY_LABEL(difficulty)}
              </span>
            </div>
          </div>
          <div className="hidden sm:block">
            <div className="text-[11px] uppercase tracking-wide text-zinc-500">
              Recent
            </div>
            <div className="flex items-center gap-1">
              {history.slice(-10).map((h, i) => (
                <span
                  key={i}
                  title={h.correct ? "correct" : "wrong"}
                  className={`inline-block h-3 w-3 rounded-sm ${
                    h.correct
                      ? "bg-emerald-500 dark:bg-emerald-400"
                      : "bg-rose-500 dark:bg-rose-400"
                  }`}
                />
              ))}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={reset}
            className="rounded border border-zinc-300 bg-white px-3 py-1 text-xs text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
          >
            Reset
          </button>
          <button
            onClick={exitToPicker}
            className="rounded border border-zinc-300 bg-white px-3 py-1 text-xs text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
          >
            Change quiz
          </button>
        </div>
      </div>

      <div className="rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="mb-3 flex items-center gap-2">
          <span
            className={`rounded px-2 py-0.5 text-[11px] font-semibold ${DIFFICULTY_COLOR(current.difficulty)}`}
          >
            {DIFFICULTY_LABEL(current.difficulty)} · {current.difficulty}/10
          </span>
          <span className="font-mono text-[11px] text-zinc-500">
            {current.nodeSlug}
          </span>
          <Link
            href={`/arch/${current.nodeSlug}`}
            target="_blank"
            rel="noopener"
            className="ml-auto inline-flex items-center gap-1 rounded border border-zinc-300 bg-white px-2 py-0.5 text-[11px] font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
            title="Open the architecture node drill-in in a new tab"
          >
            📖 Study ↗
          </Link>
        </div>
        <h2 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
          {current.question}
        </h2>

        <ul className="mt-4 space-y-2">
          {current.options.map((opt, i) => {
            const isCorrect = i === current.correctIndex;
            const isPicked = picked === i;
            let style =
              "border-zinc-200 hover:border-zinc-400 dark:border-zinc-700 dark:hover:border-zinc-500";
            if (answered) {
              if (isCorrect)
                style =
                  "border-emerald-500 bg-emerald-50 dark:bg-emerald-950/40";
              else if (isPicked && !isCorrect)
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

        {latest ? (
          <div
            className={`mt-4 rounded border p-3 text-sm ${
              latest.correct
                ? "border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-100"
                : "border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-100"
            }`}
          >
            <div className="font-semibold">
              {latest.correct ? "Correct." : "Not quite."}
            </div>
            {latest.question.explanation ? (
              <p className="mt-1 text-xs">{latest.question.explanation}</p>
            ) : null}
            {nodeForLatest ? (
              <button
                onClick={() => setShowNodeCard((v) => !v)}
                className="mt-2 inline-block text-xs underline"
              >
                {showNodeCard ? "Hide" : "Review"} the {nodeForLatest.label} node{" "}
                {showNodeCard ? "↑" : "↓"}
              </button>
            ) : null}
          </div>
        ) : null}

        {nodeForLatest && showNodeCard ? (
          <div className="mt-3 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
            <div className="flex items-baseline gap-2">
              <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                {nodeForLatest.label}
              </h3>
              <span
                className={`rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${GROUP_META[nodeForLatest.group].chip}`}
              >
                {GROUP_META[nodeForLatest.group].label}
              </span>
            </div>
            <p className="mt-2 text-sm text-zinc-800 dark:text-zinc-200">
              {nodeForLatest.blurb}
            </p>
            {nodeForLatest.repoPaths && nodeForLatest.repoPaths.length > 0 ? (
              <ul className="mt-3 flex flex-wrap gap-2 text-xs">
                {nodeForLatest.repoPaths.map((p) => (
                  <li
                    key={p.path}
                    className="rounded border border-zinc-300 bg-zinc-50 px-2 py-0.5 font-mono text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
                  >
                    {p.path}
                    {p.note ? (
                      <span className="ml-1 font-sans text-zinc-500">
                       . {p.note}
                      </span>
                    ) : null}
                  </li>
                ))}
              </ul>
            ) : null}
            {nodeForLatest.relatedModes && nodeForLatest.relatedModes.length > 0 ? (
              <div className="mt-3 flex flex-wrap gap-2 text-xs">
                {nodeForLatest.relatedModes.map((m) => (
                  <Link
                    key={m.href}
                    href={m.href}
                    target="_blank"
                    rel="noopener"
                    className="rounded border border-zinc-300 bg-white px-2 py-0.5 text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
                  >
                    {m.label} ↗
                  </Link>
                ))}
              </div>
            ) : null}
            <div className="mt-3">
              <Link
                href={`/arch/${nodeForLatest.id}`}
                target="_blank"
                rel="noopener"
                className="text-xs text-zinc-500 underline hover:text-zinc-800 dark:hover:text-zinc-200"
              >
                Open full node in a new tab ↗
              </Link>
            </div>
          </div>
        ) : null}

        <div className="mt-5 flex items-center justify-between">
          <div className="text-xs text-zinc-500">
            {answered
              ? latest?.correct
                ? "Next question will be harder."
                : "Next question will be easier."
              : picked == null
                ? "Pick an answer."
                : "Submit when ready."}
          </div>
          {answered ? (
            <button
              onClick={next}
              className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
            >
              Next question →
            </button>
          ) : (
            <button
              onClick={submit}
              disabled={picked == null}
              className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
            >
              Submit
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
