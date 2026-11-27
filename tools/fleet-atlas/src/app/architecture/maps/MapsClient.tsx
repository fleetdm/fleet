"use client";

import { useState } from "react";
import ArchGraph from "@/components/arch/ArchGraph";
import Legend from "@/components/arch/Legend";
import { SYSTEM_NODES, SYSTEM_EDGES } from "@/components/arch/system-graph";
import { SOFTWARE_NODES, SOFTWARE_EDGES } from "@/components/arch/software-graph";
import { MDM_NODES, MDM_EDGES } from "@/components/arch/mdm-graph";
import { POLICIES_NODES, POLICIES_EDGES } from "@/components/arch/policies-graph";
import { REPORTS_NODES, REPORTS_EDGES } from "@/components/arch/reports-graph";
import { GITOPS_NODES, GITOPS_EDGES } from "@/components/arch/gitops-graph";
import { MAP_BUCKETS } from "@/components/arch/map-questions";
import AssessmentPanel from "@/components/start/AssessmentPanel";
import type { AssessmentRef } from "@/components/start/paths";
import { FLEET_REPO_PATH } from "@/lib/config";

type View = "system" | "software" | "mdm" | "policies" | "reports" | "gitops";

const VIEWS: { id: View; label: string; desc: string }[] = [
  {
    id: "system",
    label: "System",
    desc: "What Fleet is made of as a deployed thing: processes, external services, data stores.",
  },
  {
    id: "software",
    label: "Software installs",
    desc: "Installs, uninstalls, catalogs (custom / FMA / VPP / in-house), vulnerability pipeline, auto-patching feedback.",
  },
  {
    id: "mdm",
    label: "MDM",
    desc: "Enrollment → protocol endpoints → config sources → reconciler crons → command queues → push → device execution → status.",
  },
  {
    id: "policies",
    label: "Policies",
    desc: "Policy creation → osquery distribution → host evaluation → membership → aggregates → automations → actions.",
  },
  {
    id: "reports",
    label: "Reports & live queries",
    desc: "Saved reports and ad-hoc live queries. Config-pull vs distributed-pull; MySQL persistence vs Redis pubsub fan-out.",
  },
  {
    id: "gitops",
    label: "GitOps",
    desc: "How declarative YAML becomes Fleet state: fleetctl parses + validates, batch endpoints receive, service diffs desired-vs-current, MySQL writes trigger reconciler crons and destructive removals.",
  },
];

const DATA: Record<
  View,
  { nodes: typeof SYSTEM_NODES; edges: typeof SYSTEM_EDGES }
> = {
  system: { nodes: SYSTEM_NODES, edges: SYSTEM_EDGES },
  software: { nodes: SOFTWARE_NODES, edges: SOFTWARE_EDGES },
  mdm: { nodes: MDM_NODES, edges: MDM_EDGES },
  policies: { nodes: POLICIES_NODES, edges: POLICIES_EDGES },
  reports: { nodes: REPORTS_NODES, edges: REPORTS_EDGES },
  gitops: { nodes: GITOPS_NODES, edges: GITOPS_EDGES },
};

type Props = { connected: boolean };

// Every map view has a matching bucket in MAP_BUCKETS, so every tab gets a
// Quiz indicator.
const TABS_WITH_QUIZ = new Set<View>(
  MAP_BUCKETS.map((b) => b.slug as View),
);

function assessmentForView(view: View): AssessmentRef[] {
  const bucket = MAP_BUCKETS.find((b) => b.slug === view);
  if (!bucket) return [];
  return [
    {
      surface: "architecture" as const,
      bucketSlug: bucket.slug,
      bucketLabel: `${bucket.label} map`,
    },
  ];
}

export default function MapsClient({ connected }: Props) {
  const [view, setView] = useState<View>("system");
  const { nodes, edges } = DATA[view];
  const desc = VIEWS.find((v) => v.id === view)!.desc;

  return (
    <div className="flex flex-col px-6 py-4">
      <header className="mb-3 pr-14">
        <div className="flex items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-1 rounded border border-zinc-200 bg-white p-1 dark:border-zinc-800 dark:bg-zinc-950">
            {VIEWS.map((v) => (
              <button
                key={v.id}
                onClick={() => setView(v.id)}
                className={`inline-flex items-center gap-1.5 rounded px-3 py-1 text-xs font-medium ${
                  view === v.id
                    ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                    : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
                }`}
              >
                {v.label}
                {TABS_WITH_QUIZ.has(v.id) ? (
                  <span
                    aria-hidden="true"
                    title="This map has an embedded quiz"
                    className={
                      view === v.id
                        ? "text-emerald-300 dark:text-emerald-600"
                        : "text-emerald-600 dark:text-emerald-400"
                    }
                  >
                    <svg
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
                  </span>
                ) : null}
              </button>
            ))}
          </div>
          <div className="flex shrink-0 items-center gap-3 text-xs">
            <span className="font-mono text-zinc-500">{FLEET_REPO_PATH}</span>
            <span
              className={
                connected
                  ? "rounded bg-emerald-100 px-2 py-0.5 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300"
                  : "rounded bg-rose-100 px-2 py-0.5 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300"
              }
            >
              {connected ? "connected" : "not found"}
            </span>
          </div>
        </div>
        <p className="mt-2 text-xs text-zinc-500">{desc}</p>
      </header>
      <div className="mb-3">
        <Legend />
      </div>
      <div className="h-[75vh] min-h-[500px]">
        <ArchGraph nodes={nodes} edges={edges} />
      </div>
      <div className="mt-6 max-w-4xl">
        <AssessmentPanel assessments={assessmentForView(view)} />
      </div>
    </div>
  );
}
