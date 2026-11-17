import Link from "next/link";
import MatchingQuiz from "@/components/matching/MatchingQuiz";

export const instant = false;

export default function MatchingPage() {
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link
          href="/self-assessments"
          className="hover:text-zinc-900 dark:hover:text-zinc-200"
        >
          ← Self-assessments
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Glossary matching
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Six terms per round, drawn from the entire Fleet glossary. Line each
          definition up with its term by dragging the right column. Rounds roll
          through the full glossary without repeating until the pool is
          exhausted.
        </p>
      </header>

      <MatchingQuiz />
    </div>
  );
}
