// Central registry of Fleet Atlas paths that embed a quiz. The nav, tab bars,
// and any card index consult this helper to decide whether to show the Quiz
// chip. Adding a new embedded quiz = one line here.
//
// Three sources are merged:
//   1. EXPLICIT_PATHS . top-level routes (not subsystems, not tracers).
//      Add a path here when you drop a quiz on a standalone page.
//   2. TRACER_PATHS   . derived from TRACERS. Every non-stub tracer has an
//      embedded TracerStepQuiz on its detail page, so this stays
//      self-syncing: add a tracer to TRACERS and its path lights up
//      automatically.
//   3. subsystem-level. Delegated to subsystemHasEmbeddedQuiz(slug), which
//      reads the SUBSYSTEM_EMBEDDED_QUIZZES map.

import { TRACERS } from "@/components/tracer/tracers";
import { subsystemHasEmbeddedQuiz } from "@/components/subsystems/embedded-quizzes";

const EXPLICIT_PATHS: ReadonlySet<string> = new Set([
  "/background/cron",
  "/background/queues",
]);

const TRACER_PATHS: ReadonlySet<string> = new Set(
  TRACERS.filter((t) => !t.stub).map((t) => `/tracer/${t.slug}`),
);

export function pathHasEmbeddedQuiz(path: string): boolean {
  if (EXPLICIT_PATHS.has(path)) return true;
  if (TRACER_PATHS.has(path)) return true;
  const subMatch = path.match(/^\/subsystems\/([^/#?]+)/);
  if (subMatch) return subsystemHasEmbeddedQuiz(subMatch[1]);
  return false;
}
