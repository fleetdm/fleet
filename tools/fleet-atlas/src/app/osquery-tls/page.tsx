import OsqueryTLSMatchingQuiz from "@/components/osquery-tls/OsqueryTLSMatchingQuiz";

export default function OsqueryTLSPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-100">
          osquery TLS endpoints
        </h1>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          The six HTTP endpoints every fleetd host talks to. Enrollment,
          scheduled config, live queries, logs, and file carves.
        </p>
      </div>
      <OsqueryTLSMatchingQuiz />
    </div>
  );
}
