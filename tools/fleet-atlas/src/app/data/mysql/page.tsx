import Link from "next/link";
import { MYSQL_GROUPS } from "@/components/data/mysql-groups";
import { editorLink } from "@/lib/config";

export const instant = false;

export default function MysqlPage() {
  const totalTables = MYSQL_GROUPS.reduce((n, g) => n + g.tables.length, 0);
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link href="/data" className="hover:text-zinc-900 dark:hover:text-zinc-200">
          ← Data layer
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          MySQL
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet&apos;s source of truth. {totalTables} tables grouped by topic.
          Schema at{" "}
          <a
            href={editorLink("server/datastore/mysql/schema.sql")}
            className="underline hover:text-zinc-900 dark:hover:text-zinc-200"
          >
            server/datastore/mysql/schema.sql
          </a>
          .
        </p>
      </header>

      <nav className="mb-10 flex flex-wrap gap-2 text-xs">
        {MYSQL_GROUPS.map((g) => (
          <a
            key={g.slug}
            href={`#${g.slug}`}
            className="rounded border border-zinc-300 bg-white px-2 py-1 text-zinc-700 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300"
          >
            {g.label} <span className="text-zinc-500">({g.tables.length})</span>
          </a>
        ))}
      </nav>

      {MYSQL_GROUPS.map((g) => (
        <section
          key={g.slug}
          id={g.slug}
          className="mb-12 scroll-mt-6 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
        >
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            {g.label}
            <span className="ml-2 text-xs font-normal text-zinc-500">
              {g.tables.length} tables
            </span>
          </h2>
          <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
            {g.blurb}
          </p>
          <ul className="mt-4 divide-y divide-zinc-200 border-t border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {g.tables.map((t) => (
              <li key={t.name} className="py-2 text-sm">
                <code className="rounded bg-zinc-100 px-1 py-0.5 font-mono text-xs text-zinc-900 dark:bg-zinc-800 dark:text-zinc-200">
                  {t.name}
                </code>
                <span className="ml-2 text-zinc-600 dark:text-zinc-400">
                  {t.note}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
