import ReviewAIQuiz from "@/components/review-ai/ReviewAIQuiz";
import QuizFormatChip from "@/components/QuizFormatChip";

export default function ReviewAIPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <div className="mb-2 flex items-center gap-2">
          <QuizFormatChip format="diff-click" />
        </div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-100">
          Review the AI
        </h1>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          AI drafts a Fleet-shaped artifact. Click the line where it got
          something wrong. Each question teaches a Fleet-specific judgment call
          AI commonly fumbles, grounded in CLAUDE.md conventions.
        </p>
      </div>
      <ReviewAIQuiz />
    </div>
  );
}
