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
import { Fragment, useMemo, useState } from "react";
import QuizFormatChip from "@/components/QuizFormatChip";
import { useIsHydrated } from "@/lib/use-is-hydrated";
import { WEBHOOK_TYPES } from "./pairs";

const STORAGE_KEY = "fleet-atlas:webhook-types-matching-quiz-best";
const IN_PROGRESS_KEY = "fleet-atlas:webhook-types-matching-quiz-in-progress";

type BestRecord = {
  bestCorrect: number;
  bestTotal: number;
  bestAttempts: number;
  bestTakenAt: number;
};

type InProgress = {
  profileOrder: string[];
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

export default function WebhookTypesMatchingQuiz() {
  const typeKeys = useMemo(() => WEBHOOK_TYPES.map((t) => t.key), []);
  const typeByKey = useMemo(() => {
    const m: Record<string, (typeof WEBHOOK_TYPES)[number]> = {};
    for (const t of WEBHOOK_TYPES) m[t.key] = t;
    return m;
  }, []);

  const hydrated = useIsHydrated();
  const [initial] = useState(() => {
    const ip = readInProgress();
    if (ip && ip.profileOrder.length === typeKeys.length) {
      return { profileOrder: ip.profileOrder, submitted: ip.submitted };
    }
    return {
      profileOrder: typeof window === "undefined" ? [] : shuffle(typeKeys),
      submitted: false,
    };
  });
  const [profileOrder, setProfileOrder] = useState<string[]>(initial.profileOrder);
  const [submitted, setSubmitted] = useState(initial.submitted);
  const [best, setBest] = useState<BestRecord | null>(() => readBest());
  const [attempts, setAttempts] = useState(0);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const perRowCorrect = profileOrder.map((key, i) => key === typeKeys[i]);
  const correctCount = perRowCorrect.filter(Boolean).length;
  const total = typeKeys.length;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id || submitted) return;
    setProfileOrder((cur) => {
      const oldIdx = cur.findIndex((s) => s === active.id);
      const newIdx = cur.findIndex((s) => s === over.id);
      const next = arrayMove(cur, oldIdx, newIdx);
      writeInProgress({ profileOrder: next, submitted: false });
      return next;
    });
  };

  const submit = () => {
    if (submitted) return;
    const newAttempts = attempts + 1;
    setAttempts(newAttempts);
    setSubmitted(true);
    writeInProgress({ profileOrder, submitted: true });

    const curBest = readBest();
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
      writeBest(rec);
      setBest(rec);
    }
  };

  const restart = () => {
    writeInProgress(null);
    setAttempts(0);
    setSubmitted(false);
    setProfileOrder(shuffle(typeKeys));
  };

  const resetBest = () => {
    if (!confirm("Reset your best score for this quiz?")) return;
    writeBest(null);
    setBest(null);
  };

  if (!hydrated || profileOrder.length === 0) return null;

  return (
    <section className="rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="matching" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Webhook types. Type / profile matching
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Drag the profile cards on the right so each lines up with the correct
        webhook type on the left. Each profile answers three questions: when it
        fires, how often Fleet checks, and how many requests go out.
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
          <SortableContext
            items={profileOrder}
            strategy={verticalListSortingStrategy}
          >
            <div className="grid grid-cols-[1fr_1.8fr] gap-3">
              {typeKeys.map((typeKey, i) => {
                const type = typeByKey[typeKey];
                const profileKey = profileOrder[i];
                const profileType = typeByKey[profileKey];
                const isCorrect = submitted ? perRowCorrect[i] : null;
                return (
                  <Fragment key={`row-${i}`}>
                    <TypeCell
                      position={i}
                      name={type.name}
                      correct={isCorrect}
                    />
                    <ProfileCard
                      slug={profileKey}
                      position={i}
                      text={profileType.profile}
                      submitted={submitted}
                      correct={isCorrect}
                    />
                  </Fragment>
                );
              })}
            </div>
          </SortableContext>
        </DndContext>

        <div className="mt-5 flex items-center justify-between">
          <div className="text-xs text-zinc-500">
            {submitted
              ? `${correctCount}/${total} correct.`
              : "Drag profiles to re-order. Tab + Space activates keyboard drag."}
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
                onClick={restart}
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

function TypeCell({
  position,
  name,
  correct,
}: {
  position: number;
  name: string;
  correct: boolean | null;
}) {
  const border =
    correct === null
      ? "border-zinc-200 dark:border-zinc-800"
      : correct
        ? "border-emerald-400 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/40"
        : "border-rose-400 bg-rose-50 dark:border-rose-700 dark:bg-rose-950/40";
  return (
    <div className={`flex items-start gap-2 rounded border p-3 ${border}`}>
      <span className="mt-0.5 font-mono text-xs text-zinc-500 w-5 text-right">
        {position + 1}.
      </span>
      <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
        {name}
      </div>
    </div>
  );
}

function ProfileCard({
  slug,
  position,
  text,
  submitted,
  correct,
}: {
  slug: string;
  position: number;
  text: string;
  submitted: boolean;
  correct: boolean | null;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: slug, disabled: submitted });
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
  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`flex cursor-grab items-start gap-2 rounded border p-3 active:cursor-grabbing ${border}`}
    >
      <span className="mt-0.5 font-mono text-xs text-zinc-500 w-5 text-right">
        {position + 1}.
      </span>
      <div className="text-sm text-zinc-800 dark:text-zinc-200">{text}</div>
    </div>
  );
}
