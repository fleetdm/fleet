// Small chip rendered on cards for pages that have an on-page assessment
// (quiz embedded in the content, visible after scrolling / navigating). The
// user sees it at a glance and knows "this page has something to prove I know
// it," without needing to click through.

export default function AssessmentIcon({ className = "" }: { className?: string }) {
  return (
    <span
      title="This page has an embedded assessment"
      className={`inline-flex items-center gap-1 rounded border border-emerald-300 bg-emerald-50 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-200 ${className}`}
    >
      <svg
        aria-hidden="true"
        width="10"
        height="10"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M9 11l3 3L22 4" />
        <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
      </svg>
      Quiz
    </span>
  );
}
