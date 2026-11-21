"use client";

import { useState } from "react";
import ArchGraph from "@/components/arch/ArchGraph";
import Legend from "@/components/arch/Legend";
import { SYSTEM_NODES, SYSTEM_EDGES } from "@/components/arch/system-graph";
import { SOFTWARE_NODES, SOFTWARE_EDGES } from "@/components/arch/software-graph";
import { MDM_NODES, MDM_EDGES } from "@/components/arch/mdm-graph";
import { POLICIES_NODES, POLICIES_EDGES } from "@/components/arch/policies-graph";
import { REPORTS_NODES, REPORTS_EDGES } from "@/components/arch/reports-graph";
import { FLEET_REPO_PATH } from "@/lib/config";

type Props = { connected: boolean };

type View = "system" | "software" | "mdm" | "policies" | "reports";

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
    desc: "Enrollment → protocol endpoints → config sources → reconciler crons → command queues → push → device execution → status write-back.",
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
];

const DATA: Record<View, { nodes: typeof SYSTEM_NODES; edges: typeof SYSTEM_EDGES }> = {
  system: { nodes: SYSTEM_NODES, edges: SYSTEM_EDGES },
  software: { nodes: SOFTWARE_NODES, edges: SOFTWARE_EDGES },
  mdm: { nodes: MDM_NODES, edges: MDM_EDGES },
  policies: { nodes: POLICIES_NODES, edges: POLICIES_EDGES },
  reports: { nodes: REPORTS_NODES, edges: REPORTS_EDGES },
};

export default function ArchPage({ connected }: Props) {
  const [view, setView] = useState<View>("system");
  const { nodes, edges } = DATA[view];
  const desc = VIEWS.find((v) => v.id === view)!.desc;

  return (
    <div className="flex h-screen flex-col bg-zinc-50 px-6 py-5 dark:bg-zinc-900">
      <header className="mb-3 flex items-start justify-between gap-6 pr-14">
        <div className="min-w-0 flex-1">
          <h1 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            Architecture map
          </h1>
          <p className="text-xs text-zinc-500">
            {desc} Press{" "}
            <kbd className="rounded border border-zinc-300 bg-white px-1 font-mono text-[10px] text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300">
              ⌘K
            </kbd>{" "}
            to search.
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-3 text-xs">
          <div className="flex items-center gap-1 rounded border border-zinc-200 bg-white p-1 dark:border-zinc-800 dark:bg-zinc-950">
            {VIEWS.map((v) => (
              <button
                key={v.id}
                onClick={() => setView(v.id)}
                className={`rounded px-3 py-1 text-xs font-medium ${
                  view === v.id
                    ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                    : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
                }`}
              >
                {v.label}
              </button>
            ))}
          </div>
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
      </header>
      <div className="mb-3">
        <Legend />
      </div>
      <ArchGraph nodes={nodes} edges={edges} />
    </div>
  );
}
