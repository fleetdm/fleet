"use client";

import {
  DndContext,
  type DragEndEvent,
  PointerSensor,
  KeyboardSensor,
  closestCenter,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
  arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMemo, useState } from "react";
import QuizFormatChip from "@/components/QuizFormatChip";
import { useIsHydrated } from "@/lib/use-is-hydrated";

export type SortStep = {
  key: string;
  // A well-known actor name. ACTOR_COLOR maps supported names to Tailwind
  // chip classes; unknown names fall back to the default.
  actor: string;
  text: string;
};

type Props = {
  title: string;
  description: string;
  storageKeyBest: string;
  storageKeyInProgress: string;
  steps: SortStep[];
};

type BestRecord = {
  bestCorrect: number;
  bestTotal: number;
  bestAttempts: number;
  bestTakenAt: number;
};

type InProgress = {
  order: string[];
  attempts: number;
  submitted: boolean;
};

function shuffle<T>(arr: T[]): T[] {
  const out = [...arr];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

function readBest(key: string): BestRecord | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as BestRecord) : null;
  } catch {
    return null;
  }
}

function writeBest(key: string, r: BestRecord | null) {
  if (typeof window === "undefined") return;
  try {
    if (r === null) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(r));
  } catch {
    /* quota */
  }
}

function readInProgress(key: string): InProgress | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as InProgress) : null;
  } catch {
    return null;
  }
}

function writeInProgress(key: string, ip: InProgress | null) {
  if (typeof window === "undefined") return;
  try {
    if (ip === null) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(ip));
  } catch {
    /* quota */
  }
}

// Shared actor-color palette across all flow sorts. Keep names short and
// stable so authors reach for the same ones; unknown actors fall back to
// a neutral chip.
const ACTOR_COLOR: Record<string, string> = {
  Admin: "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  Fleet: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  Device: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  Apple: "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  Google: "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200",
  IdP: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  External: "bg-zinc-200 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200",
};

const FALLBACK_ACTOR_CLASS =
  "bg-zinc-200 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200";

export default function SortQuiz({
  title,
  description,
  storageKeyBest,
  storageKeyInProgress,
  steps,
}: Props) {
  const canonicalKeys = useMemo(() => steps.map((s) => s.key), [steps]);
  const stepByKey = useMemo(() => {
    const m: Record<string, SortStep> = {};
    for (const s of steps) m[s.key] = s;
    return m;
  }, [steps]);

  const hydrated = useIsHydrated();
  const [initial] = useState(() => {
    const ip = readInProgress(storageKeyInProgress);
    if (ip && ip.order.length === canonicalKeys.length) {
      return { order: ip.order, attempts: ip.attempts, submitted: ip.submitted };
    }
    return {
      order: typeof window === "undefined" ? [] : shuffle(canonicalKeys),
      attempts: 0,
      submitted: false,
    };
  });
  const [order, setOrder] = useState<string[]>(initial.order);
  const [submitted, setSubmitted] = useState(initial.submitted);
  const [attempts, setAttempts] = useState(initial.attempts);
  const [best, setBest] = useState<BestRecord | null>(() =>
    readBest(storageKeyBest),
  );

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const perRowCorrect = order.map((key, i) => key === canonicalKeys[i]);
  const correctCount = perRowCorrect.filter(Boolean).length;
  const total = canonicalKeys.length;
  const allCorrect = correctCount === total;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id || submitted) return;
    setOrder((cur) => {
      const oldIdx = cur.findIndex((k) => k === active.id);
      const newIdx = cur.findIndex((k) => k === over.id);
      const next = arrayMove(cur, oldIdx, newIdx);
      writeInProgress(storageKeyInProgress, {
        order: next,
        attempts,
        submitted: false,
      });
      return next;
    });
  };

  const submit = () => {
    if (submitted) return;
    const newAttempts = attempts + 1;
    setAttempts(newAttempts);
    setSubmitted(true);
    writeInProgress(storageKeyInProgress, {
      order,
      attempts: newAttempts,
      submitted: true,
    });

    const curBest = readBest(storageKeyBest);
    const isBetter =
      !curBest ||
      correctCount > curBest.bestCorrect ||
      (correctCount === curBest.bestCorrect && newAttempts < curBest.bestAttempts);
    if (isBetter) {
      const rec: BestRecord = {
        bestCorrect: correctCount,
        bestTotal: total,
        bestAttempts: newAttempts,
        bestTakenAt: Date.now(),
      };
      writeBest(storageKeyBest, rec);
      setBest(rec);
    }
  };

  const tryAgain = () => {
    writeInProgress(storageKeyInProgress, null);
    setSubmitted(false);
    setOrder(shuffle(canonicalKeys));
  };

  const resetBest = () => {
    if (!confirm("Reset your best score for this quiz?")) return;
    writeBest(storageKeyBest, null);
    setBest(null);
  };

  if (!hydrated || order.length === 0) return null;

  return (
    <section className="rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="sorting" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        {title}
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {description}
      </p>

      {best ? (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-2 rounded border border-emerald-300 bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-200">
            Best {best.bestCorrect}/{best.bestTotal} in {best.bestAttempts}{" "}
            attempt{best.bestAttempts === 1 ? "" : "s"}
          </span>
          <button
            onClick={resetBest}
            className="rounded border border-zinc-300 bg-white px-2 py-1 text-xs font-semibold text-zinc-700 hover:border-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          >
            Reset
          </button>
        </div>
      ) : null}

      <div className="mt-5 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={order} strategy={verticalListSortingStrategy}>
            <ol className="space-y-2">
              {order.map((key, i) => {
                const step = stepByKey[key];
                const isCorrect = submitted ? perRowCorrect[i] : null;
                return (
                  <StepCard
                    key={key}
                    id={key}
                    position={i}
                    step={step}
                    submitted={submitted}
                    correct={isCorrect}
                  />
                );
              })}
            </ol>
          </SortableContext>
        </DndContext>

        <div className="mt-5 flex items-center justify-between">
          <div className="text-xs text-zinc-500">
            {submitted
              ? allCorrect
                ? `All ${total} in the right order.`
                : `${correctCount}/${total} in the right position.`
              : "Drag steps to re-order. Tab + Space activates keyboard drag."}
          </div>
          <div className="flex items-center gap-2">
            {!submitted ? (
              <button
                onClick={submit}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Submit
              </button>
            ) : (
              <button
                onClick={tryAgain}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Try again
              </button>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

function StepCard({
  id,
  position,
  step,
  submitted,
  correct,
}: {
  id: string;
  position: number;
  step: SortStep;
  submitted: boolean;
  correct: boolean | null;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id, disabled: submitted });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  };
  const border =
    correct === null
      ? "border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950"
      : correct
        ? "border-emerald-400 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/40"
        : "border-rose-400 bg-rose-50 dark:border-rose-700 dark:bg-rose-950/40";
  const actorClass = ACTOR_COLOR[step.actor] ?? FALLBACK_ACTOR_CLASS;
  return (
    <li
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`flex cursor-grab items-start gap-3 rounded border p-3 active:cursor-grabbing ${border}`}
    >
      <span className="mt-0.5 w-5 shrink-0 text-right font-mono text-xs text-zinc-500">
        {position + 1}.
      </span>
      <div className="flex-1">
        <span
          className={`mb-1 inline-block rounded px-1.5 py-0.5 text-[10px] font-semibold ${actorClass}`}
        >
          {step.actor}
        </span>
        <div className="text-sm text-zinc-800 dark:text-zinc-200">
          {step.text}
        </div>
      </div>
    </li>
  );
}
