import ValuesQuiz from "@/components/values/ValuesQuiz";
import QuizFormatChip from "@/components/QuizFormatChip";

export default function ValuesPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <div className="mb-2 flex items-center gap-2">
          <QuizFormatChip format="recall" />
        </div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-100">
          Fleet values
        </h1>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          Fleet&apos;s five core values are a shared mindset everyone on the
          team adopts. See if you can name them all from memory.
        </p>
      </div>
      <ValuesQuiz />
    </div>
  );
}
