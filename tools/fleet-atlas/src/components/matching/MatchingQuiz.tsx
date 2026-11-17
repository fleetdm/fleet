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
import { CATEGORY_COLORS, TERMS, type GlossaryTerm } from "@/components/glossary/terms";

const STORAGE_KEY = "fleet-atlas:matching-quiz-best";
const IN_PROGRESS_KEY = "fleet-atlas:matching-quiz-in-progress";

const PAIRS_PER_ROUND = 6;

type BestRecord = {
  bestCorrect: number;
  bestTotal: number;
  bestRoundsPlayed: number;
  bestPerfectRounds: number;
  bestTakenAt: number;
};

type InProgress = {
  usedSlugs: string[];
  roundSlugs: string[]; // the 6 slugs in the current round, fixed order (left column)
  definitionOrder: string[]; // right column order of slugs; user drags this
  correctCount: number;
  roundsPlayed: number;
  perfectRounds: number;
  submitted: boolean;
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

function shuffle<T>(arr: T[]): T[] {
  const out = [...arr];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

function pickRound(usedSlugs: Set<string>): string[] {
  const pool = TERMS.filter((t) => !usedSlugs.has(t.slug));
  const target = Math.min(PAIRS_PER_ROUND, pool.length);
  if (target === 0) return [];
  // If the remaining pool is smaller than PAIRS_PER_ROUND, backfill from already-used.
  if (pool.length < PAIRS_PER_ROUND) {
    const rest = TERMS.filter((t) => usedSlugs.has(t.slug));
    return shuffle([...pool, ...shuffle(rest).slice(0, PAIRS_PER_ROUND - pool.length)])
      .slice(0, PAIRS_PER_ROUND)
      .map((t) => t.slug);
  }
  return shuffle(pool).slice(0, PAIRS_PER_ROUND).map((t) => t.slug);
}

function getInitialState() {
  const best = readBest();
  const ip = readInProgress();
  if (ip && ip.roundSlugs.length === PAIRS_PER_ROUND) {
    return {
      roundSlugs: ip.roundSlugs,
      definitionOrder: ip.definitionOrder,
      usedSlugs: new Set(ip.usedSlugs),
      correctCount: ip.correctCount,
      roundsPlayed: ip.roundsPlayed,
      perfectRounds: ip.perfectRounds,
      submitted: ip.submitted,
      best,
    };
  }
  if (typeof window === "undefined") {
    return {
      roundSlugs: [] as string[],
      definitionOrder: [] as string[],
      usedSlugs: new Set<string>(),
      correctCount: 0,
      roundsPlayed: 0,
      perfectRounds: 0,
      submitted: false,
      best,
    };
  }
  const first = pickRound(new Set());
  return {
    roundSlugs: first,
    definitionOrder: shuffle(first),
    usedSlugs: new Set<string>(),
    correctCount: 0,
    roundsPlayed: 0,
    perfectRounds: 0,
    submitted: false,
    best,
  };
}

export default function MatchingQuiz() {
  const [initial] = useState(getInitialState);
  const [roundSlugs, setRoundSlugs] = useState<string[]>(initial.roundSlugs);
  const [definitionOrder, setDefinitionOrder] = useState<string[]>(
    initial.definitionOrder,
  );
  const [usedSlugs, setUsedSlugs] = useState<Set<string>>(initial.usedSlugs);
  const [correctCount, setCorrectCount] = useState(initial.correctCount);
  const [roundsPlayed, setRoundsPlayed] = useState(initial.roundsPlayed);
  const [perfectRounds, setPerfectRounds] = useState(initial.perfectRounds);
  const [submitted, setSubmitted] = useState(initial.submitted);
  const [best, setBest] = useState<BestRecord | null>(initial.best);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const termBySlug = useMemo(() => {
    const m: Record<string, GlossaryTerm> = {};
    for (const t of TERMS) m[t.slug] = t;
    return m;
  }, []);

  const perRoundCorrect = definitionOrder.map(
    (slug, i) => slug === roundSlugs[i],
  );
  const thisRoundCorrect = perRoundCorrect.filter(Boolean).length;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id || submitted) return;
    setDefinitionOrder((cur) => {
      const oldIdx = cur.findIndex((s) => s === active.id);
      const newIdx = cur.findIndex((s) => s === over.id);
      return arrayMove(cur, oldIdx, newIdx);
    });
  };

  const submit = () => {
    if (submitted) return;
    const newCorrect = correctCount + thisRoundCorrect;
    const newRounds = roundsPlayed + 1;
    const newPerfect = perfectRounds + (thisRoundCorrect === PAIRS_PER_ROUND ? 1 : 0);
    setCorrectCount(newCorrect);
    setRoundsPlayed(newRounds);
    setPerfectRounds(newPerfect);
    setSubmitted(true);

    const newUsed = new Set([...usedSlugs, ...roundSlugs]);
    setUsedSlugs(newUsed);

    writeInProgress({
      usedSlugs: Array.from(newUsed),
      roundSlugs,
      definitionOrder,
      correctCount: newCorrect,
      roundsPlayed: newRounds,
      perfectRounds: newPerfect,
      submitted: true,
    });

    // Best bump
    const curBest = readBest();
    const newTotal = newRounds * PAIRS_PER_ROUND;
    const isBetter =
      !curBest ||
      newCorrect > curBest.bestCorrect ||
      (newCorrect === curBest.bestCorrect && newTotal > curBest.bestTotal);
    if (isBetter) {
      const rec: BestRecord = {
        bestCorrect: newCorrect,
        bestTotal: newTotal,
        bestRoundsPlayed: newRounds,
        bestPerfectRounds: newPerfect,
        bestTakenAt: Date.now(),
      };
      writeBest(rec);
      setBest(rec);
    }
  };

  const nextRound = () => {
    const candidate = pickRound(usedSlugs);
    // When the pool is exhausted, reset usedSlugs and draw fresh. Otherwise
    // reuse the candidate. Compute the shuffled definition order once so the
    // UI and the persisted state agree.
    const poolExhausted = candidate.length < PAIRS_PER_ROUND;
    const nextUsed = poolExhausted ? new Set<string>() : usedSlugs;
    const nextSlugs = poolExhausted ? pickRound(new Set()) : candidate;
    const nextDefinitionOrder = shuffle(nextSlugs);

    setUsedSlugs(nextUsed);
    setRoundSlugs(nextSlugs);
    setDefinitionOrder(nextDefinitionOrder);
    setSubmitted(false);

    writeInProgress({
      usedSlugs: Array.from(nextUsed),
      roundSlugs: nextSlugs,
      definitionOrder: nextDefinitionOrder,
      correctCount,
      roundsPlayed,
      perfectRounds,
      submitted: false,
    });
  };

  const restart = () => {
    writeInProgress(null);
    setUsedSlugs(new Set());
    setCorrectCount(0);
    setRoundsPlayed(0);
    setPerfectRounds(0);
    setSubmitted(false);
    const first = pickRound(new Set());
    setRoundSlugs(first);
    setDefinitionOrder(shuffle(first));
  };

  const resetBest = () => {
    if (!confirm("Reset your best score for this quiz?")) return;
    writeBest(null);
    setBest(null);
  };

  if (roundSlugs.length === 0) {
    return (
      <section className="rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
        <div className="flex items-center gap-2">
          <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
            Prove you know it
          </h3>
          <QuizFormatChip format="matching" />
        </div>
        <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Loading...
        </h2>
      </section>
    );
  }

  return (
    <section className="rounded border border-emerald-200 bg-emerald-50/40 p-6 dark:border-emerald-800 dark:bg-emerald-950/20">
      <div className="flex items-center gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-emerald-700 dark:text-emerald-400">
          Prove you know it
        </h3>
        <QuizFormatChip format="matching" />
      </div>
      <h2 className="mt-1 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Glossary. Term / definition matching
      </h2>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {PAIRS_PER_ROUND} terms per round. Drag the definitions on the right so
        each one lines up with the correct term on the left. Tab to a definition
        and press Space to pick it up with the keyboard, then arrow keys to
        move; Space again to drop.
      </p>

      {best && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-2 rounded border border-emerald-300 bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-200">
            Best {best.bestCorrect}/{best.bestTotal} ({best.bestPerfectRounds} perfect round{best.bestPerfectRounds === 1 ? "" : "s"})
          </span>
          <button
            onClick={resetBest}
            className="rounded border border-zinc-300 bg-white px-2 py-1 text-xs font-semibold text-zinc-700 hover:border-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          >
            Reset
          </button>
        </div>
      )}

      <div className="mt-5 flex items-center justify-between text-xs text-zinc-600 dark:text-zinc-400">
        <span>
          Round {roundsPlayed + (submitted ? 0 : 1)} · {correctCount} pairs matched
          correctly total
        </span>
        <span>
          {perfectRounds} perfect round{perfectRounds === 1 ? "" : "s"}
        </span>
      </div>

      <div className="mt-4 rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={definitionOrder}
            strategy={verticalListSortingStrategy}
          >
            <div className="grid grid-cols-[1fr_1.5fr] gap-3">
              {roundSlugs.map((termSlug, i) => {
                const term = termBySlug[termSlug];
                const defSlug = definitionOrder[i];
                const defTerm = termBySlug[defSlug];
                const isCorrect = submitted ? perRoundCorrect[i] : null;
                return (
                  <Fragment key={`row-${i}`}>
                    <TermCell
                      position={i}
                      term={term}
                      correct={isCorrect}
                    />
                    <DefinitionCard
                      slug={defSlug}
                      position={i}
                      text={defTerm.short}
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
              ? `${thisRoundCorrect}/${PAIRS_PER_ROUND} correct this round.`
              : "Drag definitions to re-order. Tab + Space activates keyboard drag."}
          </div>
          <div className="flex items-center gap-2">
            {!submitted && (
              <button
                onClick={submit}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Submit
              </button>
            )}
            {submitted && (
              <button
                onClick={nextRound}
                className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                Next round →
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

function TermCell({
  position,
  term,
  correct,
}: {
  position: number;
  term: GlossaryTerm;
  correct: boolean | null;
}) {
  const border =
    correct === null
      ? "border-zinc-200 dark:border-zinc-800"
      : correct
        ? "border-emerald-400 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/40"
        : "border-rose-400 bg-rose-50 dark:border-rose-700 dark:bg-rose-950/40";
  return (
    <div
      className={`flex items-start gap-2 rounded border p-3 ${border}`}
    >
      <span className="mt-0.5 font-mono text-xs text-zinc-500 w-5 text-right">
        {position + 1}.
      </span>
      <div className="flex-1">
        <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
          {term.term}
        </div>
        {term.aka && term.aka.length > 0 ? (
          <div className="text-[11px] text-zinc-500">aka {term.aka[0]}</div>
        ) : null}
        <span
          className={`mt-1 inline-block rounded px-1.5 py-0.5 text-[10px] font-semibold ${CATEGORY_COLORS[term.category]}`}
        >
          {term.category}
        </span>
      </div>
    </div>
  );
}

function DefinitionCard({
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
  // dnd-kit's default announcer reads the sortable id, which is the term slug
  // ("sso-scim"). Override with the definition text so SRs hear the content.
  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      aria-label={`Definition: ${text}`}
      className={`flex cursor-grab items-start gap-2 rounded border p-3 active:cursor-grabbing ${border}`}
    >
      <span className="mt-0.5 font-mono text-xs text-zinc-500 w-5 text-right">
        {position + 1}.
      </span>
      <div className="text-sm text-zinc-800 dark:text-zinc-200">{text}</div>
    </div>
  );
}
