import Link from "next/link";
import { notFound } from "next/navigation";
import Markdown from "@/components/Markdown";
import { getSubsystem } from "@/components/subsystems/subsystems";
import { editorLink } from "@/lib/config";
import { fleetFileExists } from "@/lib/fleet-repo";
import AssessmentPanel from "@/components/start/AssessmentPanel";
import type { AssessmentRef } from "@/components/start/paths";
import { embeddedQuizzesFor } from "@/components/subsystems/embedded-quizzes";

function assessmentsFor(slug: string): AssessmentRef[] {
  if (slug.startsWith("apple-mdm") || slug === "microsoft-mdm" || slug === "android-mdm") {
    return [
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
    ];
  }
  if (slug === "osquery-tls" || slug === "live-queries") {
    return [
      { surface: "architecture", bucketSlug: "reports", bucketLabel: "Reports & live queries map" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
    ];
  }
  if (slug === "auth-sessions" || slug === "sso-scim") {
    return [
      { surface: "glossary", bucketSlug: "security-vuln", bucketLabel: "Standards & vuln (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
    ];
  }
  if (slug === "vuln-processing") {
    return [
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "glossary", bucketSlug: "security-vuln", bucketLabel: "Standards & vuln (glossary)" },
    ];
  }
  return [
    { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
  ];
}

export const instant = false;

export default async function SubsystemDetail({
  params,
}: PageProps<"/subsystems/[slug]">) {
  const { slug } = await params;
  const sub = getSubsystem(slug);
  if (!sub) notFound();

  const paths = await Promise.all(
    sub.repoPaths.map(async (p) => ({ path: p, exists: await fleetFileExists(p) })),
  );

  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link
          href="/subsystems"
          className="hover:text-zinc-900 dark:hover:text-zinc-200"
        >
          ← Subsystems
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          {sub.label}
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">{sub.one_liner}</p>
      </header>

      <section className="mb-10">
        <Markdown>{sub.body}</Markdown>
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Key files
        </h2>
        <ul className="space-y-2">
          {paths.map((p) => (
            <li
              key={p.path}
              className="rounded border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950"
            >
              <div className="flex items-center justify-between gap-2">
                <a
                  href={editorLink(p.path)}
                  className="font-mono text-sm text-zinc-800 hover:text-black dark:text-zinc-200 dark:hover:text-white"
                >
                  {p.path}
                </a>
                <span
                  className={
                    p.exists
                      ? "rounded bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300"
                      : "rounded bg-rose-100 px-2 py-0.5 text-xs text-rose-700 dark:bg-rose-900/40 dark:text-rose-300"
                  }
                >
                  {p.exists ? "found" : "missing"}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </section>

      {embeddedQuizzesFor(slug).map(({ anchor, Component }) => (
        <section key={anchor} id={anchor} className="mt-10 scroll-mt-20">
          <Component />
        </section>
      ))}

      <AssessmentPanel assessments={assessmentsFor(slug)} />
    </div>
  );
}
