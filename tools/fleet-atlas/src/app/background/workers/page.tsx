import { WORKERS } from "@/components/background/jobs";
import { editorLink } from "@/lib/config";

export const instant = false;

export default function WorkersPage() {
  return (
    <section>
      <h2 className="mb-4 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        Worker job types ({WORKERS.length})
      </h2>
      <p className="mb-4 text-sm text-zinc-600 dark:text-zinc-400">
        Jobs sit in the{" "}
        <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">jobs</code>{" "}
        table in MySQL. The{" "}
        <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">
          integrations
        </code>{" "}
        cron (every minute, on the Cron tab) is what drains them.
      </p>
      <ul className="space-y-2">
        {WORKERS.map((w) => (
          <li
            key={w.name}
            className="rounded border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950"
          >
            <code className="font-mono text-sm text-zinc-900 dark:text-zinc-100">
              {w.name}
            </code>
            <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
              {w.purpose}
            </p>
            <a
              href={editorLink(w.file)}
              className="mt-1 inline-block font-mono text-xs text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-300"
            >
              {w.file}
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
}
