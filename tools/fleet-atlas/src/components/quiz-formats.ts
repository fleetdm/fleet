// Shared quiz-format metadata so every assessment surface advertises its
// interaction pattern consistently (on /self-assessments and in each quiz's
// own header).

import type { SurfaceKind } from "@/lib/all-buckets";

export type QuizFormat =
  | "adaptive-multiple-choice"
  | "binary"
  | "sorting"
  | "matching"
  | "diff-click"
  | "recall";

export const QUIZ_FORMAT_LABEL: Record<QuizFormat, string> = {
  "adaptive-multiple-choice": "Adaptive multiple choice",
  binary: "Binary",
  sorting: "Sorting",
  matching: "Matching",
  "diff-click": "Code review",
  recall: "Recall",
};

export const QUIZ_FORMAT_DESCRIPTION: Record<QuizFormat, string> = {
  "adaptive-multiple-choice":
    "Pick one of four. Correct answers raise the difficulty; wrong answers lower it.",
  binary: "Pick one of two options for each question.",
  sorting: "Drag the items into the correct order.",
  matching: "Pair items from two columns.",
  "diff-click":
    "AI drafts a code snippet. Click the line that's wrong, see why, learn the Fleet convention it broke.",
  recall:
    "Type the answers from memory. No multiple choice, no hints. Just recall.",
};

export const QUIZ_FORMAT_COLOR: Record<QuizFormat, string> = {
  "adaptive-multiple-choice":
    "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200",
  binary:
    "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950/60 dark:text-amber-200",
  sorting:
    "border-violet-300 bg-violet-50 text-violet-800 dark:border-violet-700 dark:bg-violet-950/60 dark:text-violet-200",
  matching:
    "border-rose-300 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950/60 dark:text-rose-200",
  "diff-click":
    "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-200",
  recall:
    "border-fuchsia-300 bg-fuchsia-50 text-fuchsia-800 dark:border-fuchsia-700 dark:bg-fuchsia-950/60 dark:text-fuchsia-200",
};

export const SURFACE_TO_FORMAT: Record<SurfaceKind, QuizFormat> = {
  glossary: "adaptive-multiple-choice",
  architecture: "adaptive-multiple-choice",
  tracer: "sorting",
  tier: "binary",
  matching: "matching",
  "review-ai": "diff-click",
  values: "recall",
  flow: "sorting",
  cron: "adaptive-multiple-choice",
};
