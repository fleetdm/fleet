import { NODES } from "@/components/arch/graph-data";
import { TRACERS } from "@/components/tracer/tracers";
import { SUBSYSTEMS } from "@/components/subsystems/subsystems";
import { SCENARIOS } from "@/components/scenarios/scenarios";
import { TERMS } from "@/components/glossary/terms";
import { ISMS } from "@/components/fleetisms/isms";
import { PATHS } from "@/components/start/paths";

export type PaletteItem = {
  group:
    | "Modes"
    | "Start here"
    | "Architecture"
    | "Tracers"
    | "Data layer"
    | "Background"
    | "Subsystems"
    | "Scenarios"
    | "Glossary"
    | "Backend conventions";
  label: string;
  hint?: string;
  href: string;
};

const MODE_ITEMS: PaletteItem[] = [
  { group: "Modes", label: "Start here", href: "/start" },
  {
    group: "Modes",
    label: "Dev setup",
    hint: "Spin up Fleet locally: prereqs, MySQL/Redis via docker-compose, TablePlus, tests.",
    href: "/dev-setup",
  },
  { group: "Modes", label: "Architecture map", href: "/architecture/maps" },
  { group: "Modes", label: "Request tracer", href: "/tracer" },
  { group: "Modes", label: "Data layer", href: "/data" },
  { group: "Modes", label: "Background work", href: "/background" },
  { group: "Modes", label: "Subsystems", href: "/subsystems" },
  { group: "Modes", label: "Scenarios", href: "/scenarios" },
  { group: "Modes", label: "Self-assessments", href: "/self-assessments" },
  {
    group: "Modes",
    label: "Architecture self-assessments",
    hint: "Map quizzes for every architecture graph.",
    href: "/architecture/self-assessments",
  },
  {
    group: "Modes",
    label: "Glossary self-assessments",
    hint: "Matching rounds across all glossary terms.",
    href: "/glossary/self-assessments",
  },
  { group: "Modes", label: "Glossary", href: "/glossary" },
  {
    group: "Modes",
    label: "Matching (term / definition)",
    hint: "Six terms per round, drag definitions to match.",
    href: "/matching",
  },
  {
    group: "Modes",
    label: "osquery TLS endpoints (matching)",
    hint: "Match the six osquery HTTP endpoints to their plain-language purposes.",
    href: "/osquery-tls",
  },
  {
    group: "Modes",
    label: "SSO SAML login (flow sort)",
    hint: "Seven steps across Admin, Fleet, IdP. Drag them into the right order. Embedded on the SSO & SCIM subsystem page.",
    href: "/subsystems/sso-scim#sso-saml-sort",
  },
  {
    group: "Modes",
    label: "Fleet Free vs Premium",
    hint: "Binary quiz: pick the tier each feature belongs to.",
    href: "/tiers",
  },
  {
    group: "Modes",
    label: "Review the AI (code review)",
    hint: "AI drafts a Fleet artifact. Click the line that first signals a coding convention mistake.",
    href: "/review-ai",
  },
  {
    group: "Modes",
    label: "Fleet values (recall quiz)",
    hint: "Type Fleet's five core values from memory. Any order, case-insensitive.",
    href: "/values",
  },
  { group: "Modes", label: "Backend conventions", href: "/backend-conventions" },
  { group: "Modes", label: "Fleet handbook bridge", href: "/handbook" },
];

const START_ITEMS: PaletteItem[] = PATHS.map((p) => ({
  group: "Start here" as const,
  label: p.label,
  hint: p.intro,
  href: `/start#${p.slug}`,
}));

const GLOSSARY_ITEMS: PaletteItem[] = TERMS.map((t) => ({
  group: "Glossary" as const,
  label: t.term + (t.aka?.length ? ` (${t.aka.join(" / ")})` : ""),
  hint: t.short,
  href: `/glossary/terms#${t.slug}`,
}));

const ISM_ITEMS: PaletteItem[] = ISMS.map((i) => ({
  group: "Backend conventions" as const,
  label: i.title,
  hint: i.rule,
  href: `/backend-conventions#${i.slug}`,
}));

const DATA_ITEMS: PaletteItem[] = [
  {
    group: "Data layer",
    label: "The 20 tables to know",
    hint: "Curated short list of the MySQL tables every debugger needs.",
    href: "/data/key-tables",
  },
  { group: "Data layer", label: "MySQL schema", href: "/data/mysql" },
  { group: "Data layer", label: "Redis uses", href: "/data/redis" },
  { group: "Data layer", label: "File store", href: "/data/filestore" },
];

const BG_ITEMS: PaletteItem[] = [
  { group: "Background", label: "Cron schedules", href: "/background/cron" },
  { group: "Background", label: "Worker queue", href: "/background/workers" },
  { group: "Background", label: "Queues (Redis + DB)", href: "/background/queues" },
];

const ARCH_ITEMS: PaletteItem[] = NODES.map((n) => ({
  group: "Architecture" as const,
  label: n.label,
  hint: n.blurb,
  href: `/arch/${n.id}`,
}));

const TRACER_ITEMS: PaletteItem[] = TRACERS.map((t) => ({
  group: "Tracers" as const,
  label: `${t.method} ${t.path}`,
  hint: t.summary,
  href: `/tracer/${t.slug}`,
}));

const SUB_ITEMS: PaletteItem[] = SUBSYSTEMS.map((s) => ({
  group: "Subsystems" as const,
  label: s.label,
  hint: s.one_liner,
  href: `/subsystems/${s.slug}`,
}));

const SCEN_ITEMS: PaletteItem[] = SCENARIOS.map((s) => ({
  group: "Scenarios" as const,
  label: s.label,
  hint: s.summary,
  href: `/scenarios/${s.slug}`,
}));

export const PALETTE_ITEMS: PaletteItem[] = [
  ...MODE_ITEMS,
  ...START_ITEMS,
  ...DATA_ITEMS,
  ...BG_ITEMS,
  ...ARCH_ITEMS,
  ...TRACER_ITEMS,
  ...SUB_ITEMS,
  ...SCEN_ITEMS,
  ...GLOSSARY_ITEMS,
  ...ISM_ITEMS,
];

export function filterItems(query: string): PaletteItem[] {
  const q = query.trim().toLowerCase();
  if (!q) return PALETTE_ITEMS;
  return PALETTE_ITEMS.filter((it) => {
    const hay = `${it.group} ${it.label} ${it.hint ?? ""}`.toLowerCase();
    return q.split(/\s+/).every((token) => hay.includes(token));
  });
}
