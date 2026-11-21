import Link from "next/link";
import { notFound } from "next/navigation";
import { TechnicalHelp } from "@/providers/TechnicalHelp";
import Markdown from "@/components/Markdown";
import { GROUP_META, getNode } from "@/components/arch/graph-data";
import { NODE_DETAILS } from "@/components/arch/node-details";
import { editorLink } from "@/lib/config";
import { fleetFileExists } from "@/lib/fleet-repo";
import AssessmentPanel from "@/components/start/AssessmentPanel";
import type { AssessmentRef } from "@/components/start/paths";

const MAP_LABEL: Record<string, string> = {
  system: "System map",
  software: "Software installs map",
  mdm: "MDM map",
  policies: "Policies map",
  reports: "Reports & live queries map",
  gitops: "GitOps map",
};

function archBucketFor(nodeId: string): AssessmentRef[] {
  const match =
    nodeId.startsWith("sys-")
      ? "system"
      : nodeId.startsWith("sw-")
        ? "software"
        : nodeId.startsWith("mdm-")
          ? "mdm"
          : nodeId.startsWith("pol-")
            ? "policies"
            : nodeId.startsWith("rep-")
              ? "reports"
              : nodeId.startsWith("git-")
                ? "gitops"
                : null;
  if (!match) return [];
  return [
    { surface: "architecture", bucketSlug: match, bucketLabel: MAP_LABEL[match] },
  ];
}

export const instant = false;

export default async function ArchNodePage({
  params,
}: PageProps<"/arch/[id]">) {
  const { id } = await params;
  const node = getNode(id);
  if (!node) notFound();

  const meta = GROUP_META[node.group];

  const pathsWithExistence = await Promise.all(
    node.repoPaths.map(async (p) => ({
      ...p,
      exists: await fleetFileExists(p.path),
    })),
  );

  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link
          href="/architecture/maps"
          className="hover:text-zinc-900 dark:hover:text-zinc-200"
        >
          ← Architecture map
        </Link>
      </nav>
      <div className={`mb-6 rounded border ${meta.card} p-4`}>
        <div className="text-xs uppercase tracking-widest text-zinc-600 dark:text-zinc-400">
          {meta.label}
        </div>
        <h1 className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          {node.label}
        </h1>
      </div>

      <section className="mb-8">
        <p className="text-base leading-relaxed text-zinc-700 dark:text-zinc-300">
          {node.blurb}
        </p>
      </section>

      {NODE_DETAILS[node.id] ? (
        <section className="mb-10">
          <Markdown>{NODE_DETAILS[node.id]}</Markdown>
        </section>
      ) : null}

      <section className="mb-8">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Fleet repo paths
        </h2>
        <ul className="space-y-2">
          {pathsWithExistence.map((p) => (
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
              {p.note ? (
                <div className="mt-1 text-xs text-zinc-500">{p.note}</div>
              ) : null}
            </li>
          ))}
        </ul>
      </section>

      {node.technicalHelp ? (
        <section className="mb-8">
          <TechnicalHelp
            technical={node.technicalHelp.technical}
            analogies={node.technicalHelp.analogies}
          />
        </section>
      ) : null}

      {node.relatedModes && node.relatedModes.length > 0 ? (
        <section>
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
            Keep going
          </h2>
          <ul className="space-y-2">
            {node.relatedModes.map((r) => (
              <li key={r.href}>
                <Link
                  href={r.href}
                  className="inline-block rounded border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-800 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800"
                >
                  {r.label} →
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {archBucketFor(id).length > 0 ? (
        <AssessmentPanel assessments={archBucketFor(id)} />
      ) : null}
    </div>
  );
}
