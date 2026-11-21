import {
  QUIZ_FORMAT_COLOR,
  QUIZ_FORMAT_DESCRIPTION,
  QUIZ_FORMAT_LABEL,
  type QuizFormat,
} from "./quiz-formats";

export default function QuizFormatChip({ format }: { format: QuizFormat }) {
  return (
    <span
      title={QUIZ_FORMAT_DESCRIPTION[format]}
      className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${QUIZ_FORMAT_COLOR[format]}`}
    >
      {QUIZ_FORMAT_LABEL[format]}
    </span>
  );
}
