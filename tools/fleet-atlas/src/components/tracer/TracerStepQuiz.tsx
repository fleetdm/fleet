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
import type { TracerStep } from "./tracers";

const STORAGE_KEY = "fleet-atlas:tracer-quiz-best";

type BestRecord = {
  bestCorrect: number;
  total: number;
  bestAttempts: number;
  bestTakenAt: number;
};

function loadBest(slug: string): BestRecord | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const map = JSON.parse(raw) as Record<string, BestRecord>;
    return map[slug] ?? null;
  } catch {
    return null;
  }
}

function saveBest(slug: string, record: BestRecord) {
  if (typeof window === "undefined") return;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const map = raw ? (JSON.parse(raw) as Record<string, BestRecord>) : {};
    map[slug] = record;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch {
    /* ignore quota */
  }
}

const LAYER_COLORS: Record<string, string> = {
  Router:
    "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950 dark:text-sky-200",
  Decoder:
    "border-cyan-300 bg-cyan-50 text-cyan-800 dark:border-cyan-700 dark:bg-cyan-950 dark:text-cyan-200",
  Endpoint:
    "border-teal-300 bg-teal-50 text-teal-800 dark:border-teal-700 dark:bg-teal-950 dark:text-teal-200",
  Service:
    "border-indigo-300 bg-indigo-50 text-indigo-800 dark:border-indigo-700 dark:bg-indigo-950 dark:text-indigo-200",
  Authz:
    "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-200",
  Datastore:
    "border-violet-300 bg-violet-50 text-violet-800 dark:border-violet-700 dark:bg-violet-950 dark:text-violet-200",
  MySQL:
    "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950 dark:text-emerald-200",
  Response:
    "border-rose-300 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950 dark:text-rose-200",
};

type Item = {
  id: string;
  originalIndex: number;
  layer: string;
  title: string;
};

function shuffle<T>(arr: T[], seed: number): T[] {
  const out = [...arr];
  let r = seed;
  for (let i = out.length - 1; i > 0; i--) {
    r = (r * 9301 + 49297) % 233280;
    const j = Math.floor((r / 233280) * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

export default function TracerStepQuiz({
  slug,
  method,
  path,
  steps,
}: {
  slug: string;
  method: string;
  path: string;
  steps: TracerStep[];
}) {
  const initialItems = useMemo<Item[]>(() => {
    const base: Item[] = steps.map((s, i) => ({
      id: `${slug}-${i}`,
      originalIndex: i,
      layer: s.layer,
      title: s.title,
    }));
    const seed = slug.split("").reduce((a, c) => a + c.charCodeAt(0), 0);
    return shuffle(base, seed);
  }, [slug, steps]);

  const [items, setItems] = useState<Item[]>(initialItems);
  const [checked, setChecked] = useState(false);
  const [attempts, setAttempts] = useState(0);
  const [revealed, setRevealed] = useState(false);
  const [best, setBest] = useState<BestRecord | null>(() => loadBest(slug));

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setItems((cur) => {
      const oldIdx = cur.findIndex((i) => i.id === active.id);
      const newIdx = cur.findIndex((i) => i.id === over.id);
      return arrayMove(cur, oldIdx, newIdx);
    });
    if (checked) setChecked(false);
  };

  const perItemCorrect = items.map(
    (item, position) => item.originalIndex === position,
  );
  const correctCount = perItemCorrect.filter(Boolean).length;
  const allCorrect = correctCount === items.length;

  const reset = () => {
    setItems(shuffle(initialItems, Date.now()));
    setChecked(false);
    setAttempts(0);
    setRevealed(false);
  };

  const reveal = () => {
    setItems([...initialItems].sort((a, b) => a.originalIndex - b.originalIndex));
    setChecked(true);
    setRevealed(true);
  };

  const check = () => {
    const nextAttempts = attempts + 1;
    setChecked(true);
    setAttempts(nextAttempts);
    if (revealed) return;
    const isBetter =
      !best ||
      correctCount > best.bestCorrect ||
      (correctCount === best.bestCorrect && nextAttempts < best.bestAttempts);
    if (isBetter) {
      const record: BestRecord = {
        bestCorrect: correctCount,
        total: items.length,
        bestAttempts: nextAttempts,
        bestTakenAt: Date.now(),
      };
      saveBest(slug, record);
      setBest(record);
    }
  };

  return (
    <section className="mt-10 rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="sorting" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Order the steps for{" "}
        <span className="font-mono text-sm">
          {method} {path}
        </span>
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Drag each card into the order Fleet processes the request, top to bottom.
      </p>

      {best && (
        <div className="mt-3 inline-flex items-center gap-2 rounded border border-emerald-300 bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-200">
          <span>
            Best {best.bestCorrect}/{best.total}
          </span>
          {best.bestCorrect === best.total && (
            <span className="font-normal text-emerald-700 dark:text-emerald-300">
              in {best.bestAttempts} {best.bestAttempts === 1 ? "attempt" : "attempts"}
            </span>
          )}
        </div>
      )}

      <div className="mt-5">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={items.map((i) => i.id)} strategy={verticalListSortingStrategy}>
            <ol className="space-y-2">
              {items.map((item, position) => (
                <SortableCard
                  key={item.id}
                  item={item}
                  position={position}
                  checked={checked}
                  correct={perItemCorrect[position]}
                  revealed={revealed}
                />
              ))}
            </ol>
          </SortableContext>
        </DndContext>
      </div>

      <div className="mt-5 flex flex-wrap items-center gap-3">
        {!revealed && (
          <button
            onClick={check}
            className="rounded border border-zinc-900 bg-zinc-900 px-3 py-1.5 text-sm font-semibold text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            Check order
          </button>
        )}
        <button
          onClick={reset}
          className="rounded border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
        >
          Reshuffle
        </button>
        {checked && !allCorrect && attempts >= 2 && !revealed && (
          <button
            onClick={reveal}
            className="rounded border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-600 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400"
          >
            Reveal solution
          </button>
        )}

        {checked && (
          <span
            className={`text-sm font-semibold ${
              allCorrect
                ? "text-emerald-700 dark:text-emerald-400"
                : "text-zinc-600 dark:text-zinc-400"
            }`}
          >
            {allCorrect
              ? "All in order"
              : `${correctCount}/${items.length} in the right spot`}
          </span>
        )}
      </div>
    </section>
  );
}

function SortableCard({
  item,
  position,
  checked,
  correct,
  revealed,
}: {
  item: Item;
  position: number;
  checked: boolean;
  correct: boolean;
  revealed: boolean;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: item.id });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  };

  const stateBorder = checked
    ? correct
      ? "border-emerald-400 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/40"
      : "border-rose-400 bg-rose-50 dark:border-rose-700 dark:bg-rose-950/40"
    : "border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950";

  return (
    <li
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`flex cursor-grab items-center gap-3 rounded border p-3 active:cursor-grabbing ${stateBorder}`}
    >
      <span className="font-mono text-xs text-zinc-500 w-6 text-right">
        {String(position + 1).padStart(2, "0")}
      </span>
      <span
        className={`shrink-0 rounded border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${
          LAYER_COLORS[item.layer] ?? ""
        }`}
      >
        {item.layer}
      </span>
      <span className="text-sm text-zinc-900 dark:text-zinc-100">{item.title}</span>
      {revealed && (
        <span className="ml-auto text-xs font-mono text-zinc-400">
          #{String(item.originalIndex + 1).padStart(2, "0")}
        </span>
      )}
    </li>
  );
}
