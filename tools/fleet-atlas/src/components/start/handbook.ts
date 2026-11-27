export type HandbookLink = {
  title: string;
  href: string;
  blurb: string;
};

export type HandbookRole = {
  roleLabel: string;
  intro: string;
  pointers: HandbookLink[];
};

export const CROSS_CUTTING: HandbookLink[] = [
  {
    title: "Why this way",
    href: "https://fleetdm.com/handbook/company/why-this-way",
    blurb:
      "Fleet's design philosophy, in Fleet's own words. The highest-ROI doc for understanding why Fleet is built the way it is.",
  },
  {
    title: "Core values",
    href: "https://fleetdm.com/handbook/company#values",
    blurb:
      "Fleet's values. The handful of principles everything else at Fleet gets evaluated against. Read before Communications or Writing; they're the layer underneath.",
  },
  {
    title: "Company",
    href: "https://fleetdm.com/handbook/company",
    blurb:
      "Mission, purpose, culture, history. Read this before anything else.",
  },
  {
    title: "Communications",
    href: "https://fleetdm.com/handbook/company/communications",
    blurb:
      "How Fleet writes publicly. Tone, conventions, what to publish, what to keep internal.",
  },
  {
    title: "Writing",
    href: "https://fleetdm.com/handbook/company/writing",
    blurb:
      "Fleet's writing style guide. Applies to every public artifact including PRs, blog posts, and docs.",
  },
  {
    title: "Product groups",
    href: "https://fleetdm.com/handbook/company/product-groups",
    blurb:
      "The engineering + product org structure: who owns what, how teams are shaped.",
  },
  {
    title: "Leadership",
    href: "https://fleetdm.com/handbook/company/leadership",
    blurb: "Who's who at the top; decision rights and escalation paths.",
  },
];

// Keyed by the PATH / TEAM slug used in paths.ts, so Start here can look up
// handbook pointers for whichever role or team the user has selected.
export const HANDBOOK_BY_SLUG: Record<string, HandbookRole> = {
  "backend-engineer": {
    roleLabel: "Backend engineer",
    intro:
      "Atlas gives you Fleet's product internals; the handbook gives you how Fleet engineers work.",
    pointers: [
      {
        title: ".claude/rules/fleet-go-backend.md",
        href: "https://github.com/fleetdm/fleet/blob/main/.claude/rules/fleet-go-backend.md",
        blurb:
          "Fleet's Go backend conventions. Error handling (ctxerr.Wrap), input validation, service method patterns, viewer context, logging, imports. Prescriptive and short; the day-one reference for anyone writing Go in Fleet.",
      },
      {
        title: "Engineering handbook",
        href: "https://fleetdm.com/handbook/engineering",
        blurb:
          "Rituals, ownership, review conventions, deployment procedures. Required reading week one.",
      },
      {
        title: "Debugging",
        href: "https://fleetdm.com/handbook/engineering/Debugging",
        blurb:
          "Fleet's house debugging playbook. Pairs well with the data-layer and request-tracer pages here.",
      },
      {
        title: "Releases",
        href: "https://fleetdm.com/handbook/engineering/releases",
        blurb:
          "Release cadence, cutting a release candidate, cherry-pick rules, how patches ship.",
      },
      {
        title: "Scaling Fleet",
        href: "https://fleetdm.com/handbook/engineering/scaling-fleet",
        blurb:
          "What breaks first at scale, where Fleet's bottlenecks are, performance targets.",
      },
      {
        title: "Load testing",
        href: "https://fleetdm.com/handbook/engineering/Load-testing",
        blurb: "Fleet's load testing setup. Tools, scenarios, metrics watched.",
      },
      {
        title: "Runbooks",
        href: "https://fleetdm.com/handbook/engineering/runbooks",
        blurb:
          "Incident response playbooks and on-call procedures for Fleet's managed hosting.",
      },
      {
        title: "TUF",
        href: "https://fleetdm.com/handbook/engineering/tuf",
        blurb:
          "Fleet's The Update Framework story. How fleetd auto-updates work on the ops side.",
      },
    ],
  },
  "frontend-engineer": {
    roleLabel: "Frontend engineer",
    intro:
      "The product internals Atlas covers are mostly backend; the handbook has FE-specific rituals + the website / docs story. Start with the frontend README. It's the admin UI stack's actual entry point.",
    pointers: [
      {
        title: "frontend/README.md (fleetdm/fleet repo)",
        href: "https://github.com/fleetdm/fleet/blob/main/frontend/README.md",
        blurb:
          "The admin UI: React + TypeScript + Hooks SPA. Directory structure, patterns, running locally, testing strategy, Storybook. Read this day one.",
      },
      {
        title: "frontend/docs/patterns.md",
        href: "https://github.com/fleetdm/fleet/blob/main/frontend/docs/patterns.md",
        blurb:
          "Fleet's house frontend patterns. How components are structured, state management conventions, naming, styling. The playbook under the README.",
      },
      {
        title: ".claude/rules/fleet-frontend.md",
        href: "https://github.com/fleetdm/fleet/blob/main/.claude/rules/fleet-frontend.md",
        blurb:
          "The frontend conventions file Claude Code auto-loads when you edit matching files. Short, prescriptive, and easier to skim than the full README when you need a quick refresh.",
      },
      {
        title: "Engineering handbook",
        href: "https://fleetdm.com/handbook/engineering",
        blurb: "Same base as the BE path. Rituals, review conventions, releases.",
      },
      {
        title: "Website",
        href: "https://fleetdm.com/handbook/engineering/website",
        blurb:
          "fleetdm.com itself (Sails.js app). How it's built, how to contribute. Different stack than the admin UI.",
      },
      {
        title: "Product design",
        href: "https://fleetdm.com/handbook/product-design",
        blurb:
          "Design partner's rituals + handoff conventions. Important if you're building UI from Figma.",
      },
    ],
  },
  qa: {
    roleLabel: "QA",
    intro:
      "Debugging and release-engineering reading. Pairs well with Atlas's scenarios + request tracer.",
    pointers: [
      {
        title: "Engineering handbook",
        href: "https://fleetdm.com/handbook/engineering",
        blurb: "Rituals, review conventions, bug triage flow.",
      },
      {
        title: "Debugging",
        href: "https://fleetdm.com/handbook/engineering/Debugging",
        blurb: "House debugging playbook. Second only to Atlas's scenarios page.",
      },
      {
        title: "Releases",
        href: "https://fleetdm.com/handbook/engineering/releases",
        blurb: "RC process, cherry-pick rules, release validation checklist.",
      },
    ],
  },
  "infra-eng": {
    roleLabel: "Infrastructure engineer",
    intro:
      "Everything the engineering section plus the parts that cover deployment and operations.",
    pointers: [
      {
        title: ".claude/rules/fleet-database.md",
        href: "https://github.com/fleetdm/fleet/blob/main/.claude/rules/fleet-database.md",
        blurb:
          "Fleet's MySQL + migration conventions. Schema patterns, migration structure, DBReader/DBWriter rules, replica awareness. Short and prescriptive; the on-call reference for schema-level changes.",
      },
      {
        title: "Engineering handbook",
        href: "https://fleetdm.com/handbook/engineering",
        blurb: "Shared base.",
      },
      {
        title: "Scaling Fleet",
        href: "https://fleetdm.com/handbook/engineering/scaling-fleet",
        blurb: "The reference for 'can Fleet handle N hosts?' conversations.",
      },
      {
        title: "Runbooks",
        href: "https://fleetdm.com/handbook/engineering/runbooks",
        blurb: "On-call playbooks for Fleet's managed hosting.",
      },
      {
        title: "IT + security",
        href: "https://fleetdm.com/handbook/it/security",
        blurb: "Fleet's own IT security posture. What Fleet runs internally.",
      },
      {
        title: "TUF",
        href: "https://fleetdm.com/handbook/engineering/tuf",
        blurb:
          "Fleet's update framework. Relevant if you're touching fleetd auto-update infrastructure.",
      },
    ],
  },
  "product-design": {
    roleLabel: "Product design",
    intro: "Design process, handoff conventions, and Fleet's design values.",
    pointers: [
      {
        title: "Product design handbook",
        href: "https://fleetdm.com/handbook/product-design",
        blurb: "Design partner rituals, handoff process, Figma conventions.",
      },
      {
        title: "Brand",
        href: "https://fleetdm.com/handbook/company/brand",
        blurb: "Fleet's visual identity. Logo, colors, typography, usage.",
      },
      {
        title: "Writing",
        href: "https://fleetdm.com/handbook/company/writing",
        blurb:
          "UI copy conventions and tone. Applies to buttons, help text, docs.",
      },
    ],
  },
  "customer-support-sa": {
    roleLabel: "Customer support & solutions architecture",
    intro:
      "Customer-success is where day-to-day workflows live. Pairs with Atlas's debug-adjacent scenarios + the 20 tables page.",
    pointers: [
      {
        title: "Customer success handbook",
        href: "https://fleetdm.com/handbook/customer-success",
        blurb:
          "CS rituals, ticket flow, customer check-in cadence, escalation procedures.",
      },
      {
        title: "Engineering: Debugging",
        href: "https://fleetdm.com/handbook/engineering/Debugging",
        blurb:
          "When a customer ticket needs engineering-depth debugging, this is the shared playbook.",
      },
      {
        title: "Communications",
        href: "https://fleetdm.com/handbook/company/communications",
        blurb:
          "Public writing + customer-facing comms conventions. Important when drafting response templates.",
      },
    ],
  },
  "customer-success": {
    roleLabel: "Customer success",
    intro:
      "CS rituals + the broader company context on how Fleet thinks about customers.",
    pointers: [
      {
        title: "Customer success handbook",
        href: "https://fleetdm.com/handbook/customer-success",
        blurb: "Rituals, cadence, customer lifecycle procedures.",
      },
      {
        title: "Go-to-market operations",
        href: "https://fleetdm.com/handbook/company/go-to-market-operations",
        blurb: "How Fleet goes to market. Pipeline, segmentation, metrics.",
      },
      {
        title: "Communications",
        href: "https://fleetdm.com/handbook/company/communications",
        blurb: "Public writing + customer-facing comms conventions.",
      },
      {
        title: "Pricing + features",
        href: "https://fleetdm.com/handbook/company/pricing-features-table.yml",
        blurb:
          "The pricing table source of truth. Pair with Atlas's Free vs Premium page for the engineering-side gating story.",
      },
    ],
  },
  sales: {
    roleLabel: "Sales",
    intro:
      "Opportunity flow + Fleet's sales conventions. Atlas handles the technical details customers probe during qualification.",
    pointers: [
      {
        title: "Sales handbook",
        href: "https://fleetdm.com/handbook/sales",
        blurb: "Rituals, process, forecast cadence.",
      },
      {
        title: "Opportunity stages",
        href: "https://fleetdm.com/handbook/sales/opportunity-stages",
        blurb:
          "Fleet's definition of each pipeline stage; what qualifies something as closed-won.",
      },
      {
        title: "Sales policies",
        href: "https://fleetdm.com/handbook/sales/sales-policies",
        blurb: "Discounting, approvals, renewal rules.",
      },
      {
        title: "Go-to-market operations",
        href: "https://fleetdm.com/handbook/company/go-to-market-operations",
        blurb: "How Fleet goes to market end-to-end, including CS handoffs.",
      },
      {
        title: "Pricing + features",
        href: "https://fleetdm.com/handbook/company/pricing-features-table.yml",
        blurb: "Pricing table source of truth.",
      },
    ],
  },
  "it-strategy": {
    roleLabel: "IT strategy",
    intro:
      "Both the IT handbook (what Fleet does internally) and the broader business reading (what Fleet publishes externally).",
    pointers: [
      {
        title: "IT handbook",
        href: "https://fleetdm.com/handbook/it",
        blurb: "How Fleet runs its own IT. A reference for what customers do too.",
      },
      {
        title: "Security",
        href: "https://fleetdm.com/handbook/it/security",
        blurb:
          "Fleet's security posture, what Fleet protects, and how. Important for security-conscious conversations.",
      },
      {
        title: "Scaling Fleet",
        href: "https://fleetdm.com/handbook/engineering/scaling-fleet",
        blurb: "What breaks first at scale. Important for strategic conversations.",
      },
      {
        title: "Product groups",
        href: "https://fleetdm.com/handbook/company/product-groups",
        blurb:
          "Who owns which Fleet features. Useful when positioning the product externally.",
      },
    ],
  },
};

// Teams (product groups) don't have their own handbook sections, but the
// engineers staffing them want the Backend / Frontend engineer reading set.
// Map each team slug to the handbook role we show for members of that team.
export const TEAM_HANDBOOK_FALLBACK: Record<string, string> = {
  "apple-at-work": "backend-engineer",
  "power-to-pc": "backend-engineer",
  byod: "backend-engineer",
  orchestration: "backend-engineer",
  "auto-patching": "backend-engineer",
};

export function handbookRoleForSlug(slug: string): HandbookRole | null {
  if (HANDBOOK_BY_SLUG[slug]) return HANDBOOK_BY_SLUG[slug];
  const fallback = TEAM_HANDBOOK_FALLBACK[slug];
  if (fallback && HANDBOOK_BY_SLUG[fallback]) return HANDBOOK_BY_SLUG[fallback];
  return null;
}
