import { BUCKETS, poolForBucket } from "@/components/glossary/questions";
import { MAP_BUCKETS, poolForMapBucket } from "@/components/arch/map-questions";
import { TRACERS } from "@/components/tracer/tracers";
import { TIER_QUESTIONS } from "@/components/tiers/quiz-questions";
import { TERMS } from "@/components/glossary/terms";
import { REVIEW_AI_QUESTIONS } from "@/components/review-ai/questions";
import { CRON_QUESTIONS } from "@/components/background/cron-questions";

export type SurfaceKind =
  | "glossary"
  | "architecture"
  | "tracer"
  | "tier"
  | "matching"
  | "review-ai"
  | "values"
  | "flow"
  | "cron";

export type UnifiedBucket = {
  surface: SurfaceKind;
  slug: string;
  label: string;
  description: string;
  questionCount: number;
  surfaceLabel: string;
  surfaceHref: string;
};

export function allBuckets(): UnifiedBucket[] {
  const glossary = BUCKETS.map((b) => ({
    surface: "glossary" as const,
    slug: b.slug,
    label: b.label,
    description: b.description,
    questionCount: poolForBucket(b).length,
    surfaceLabel: "Glossary",
    surfaceHref: "/glossary/self-assessments",
  }));
  const arch = MAP_BUCKETS.map((b) => ({
    surface: "architecture" as const,
    slug: b.slug,
    label: b.label,
    description: b.description,
    questionCount: poolForMapBucket(b).length,
    surfaceLabel: "Architecture map",
    surfaceHref: "/architecture/self-assessments",
  }));
  const tracer = TRACERS.map((t) => ({
    surface: "tracer" as const,
    slug: t.slug,
    label: `${t.method} ${t.path}`,
    description: t.summary,
    questionCount: t.steps.length,
    surfaceLabel: "Request tracer",
    surfaceHref: "/tracer",
  }));
  const tier = [
    {
      surface: "tier" as const,
      slug: "free-vs-premium",
      label: "Fleet Free vs Fleet Premium",
      description:
        "Binary quiz: pick the tier each feature belongs to. Reveals the specific premium gate on each miss.",
      questionCount: TIER_QUESTIONS.length,
      surfaceLabel: "Tier knowledge",
      surfaceHref: "/tiers",
    },
  ];
  const matching = [
    {
      surface: "matching" as const,
      slug: "glossary-all",
      label: "Glossary: Term / definition matching",
      description:
        "Six terms per round, drawn from the full glossary. Drag definitions to match. Rounds don't repeat terms until the pool is exhausted.",
      questionCount: TERMS.length,
      surfaceLabel: "Matching",
      surfaceHref: "/matching",
    },
    {
      surface: "matching" as const,
      slug: "osquery-tls",
      label: "osquery TLS: Endpoint / purpose matching",
      description:
        "Six core osquery HTTP endpoints. Drag the plain-language descriptions to the right endpoint.",
      questionCount: 6,
      surfaceLabel: "Matching",
      surfaceHref: "/osquery-tls",
    },
    {
      surface: "matching" as const,
      slug: "webhook-types",
      label: "Webhook types: Type / profile matching",
      description:
        "Five Fleet webhook types matched to their fires-when + check cadence + requests-sent profile. Embedded on the Webhooks & integrations subsystem page.",
      questionCount: 5,
      surfaceLabel: "Matching",
      surfaceHref: "/subsystems/webhooks-integrations#webhook-types-matching",
    },
    {
      surface: "matching" as const,
      slug: "interval-defaults",
      label: "Fleet interval defaults: Knob / value matching",
      description:
        "Six Fleet knobs and their defaults: distributed_interval, detail_update_interval, policy_update_interval, result cap, splay, policy webhook check. Embedded on the Cron schedules page.",
      questionCount: 6,
      surfaceLabel: "Matching",
      surfaceHref: "/background/cron#interval-defaults-matching",
    },
    {
      surface: "matching" as const,
      slug: "queue-storage",
      label: "Fleet queues: Storage + drain matching",
      description:
        "Six queues (Worker, VPP install, Custom installer, Apple MDM command, Live query pubsub, Scheduled batch) matched to their storage shape + drain mechanism. Embedded on the All queues page.",
      questionCount: 6,
      surfaceLabel: "Matching",
      surfaceHref: "/background/queues#queue-storage-matching",
    },
  ];
  const reviewAI = [
    {
      surface: "review-ai" as const,
      slug: "all",
      label: "Review the AI: Click the line that's wrong",
      description:
        "AI drafts a Fleet-shaped artifact. Click the line where it broke a Fleet convention. Teaches the judgment AI can't do for you.",
      questionCount: REVIEW_AI_QUESTIONS.length,
      surfaceLabel: "Code review",
      surfaceHref: "/review-ai",
    },
  ];
  const values = [
    {
      surface: "values" as const,
      slug: "all",
      label: "Fleet values: Recall all 5",
      description:
        "Type Fleet's five core values from memory. Any order. Case-insensitive. Shared mindset across the whole team.",
      questionCount: 5,
      surfaceLabel: "Fleet values",
      surfaceHref: "/values",
    },
  ];
  const flow = [
    {
      surface: "flow" as const,
      slug: "sso-saml",
      label: "SSO SAML login: Admin flow order",
      description:
        "Seven steps across Admin, Fleet, and IdP. Drag them into the right order. Embedded on the SSO & SCIM subsystem page.",
      questionCount: 7,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/sso-scim#sso-saml-sort",
    },
    {
      surface: "flow" as const,
      slug: "mac-dep",
      label: "Mac DEP enrollment: Flow order",
      description:
        "Eight steps from ABM assignment to setup assistant. Embedded on the Apple MDM subsystem page.",
      questionCount: 8,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/apple-mdm#mac-dep-sort",
    },
    {
      surface: "flow" as const,
      slug: "setup-experience",
      label: "Setup experience install: Flow order",
      description:
        "Eight steps. Fleet pre-installs apps while the Mac is paused at 'awaiting configuration,' then releases via DDM. Embedded on the Apple MDM subsystem page.",
      questionCount: 8,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/apple-mdm#setup-experience-sort",
    },
    {
      surface: "flow" as const,
      slug: "live-query",
      label: "Live query: Flow order",
      description:
        "Nine steps from Run click to rows streaming in. MySQL for cross-instance campaign visibility, Redis pubsub for bridging the Fleet instance that receives host results to the one holding the admin's WebSocket. Embedded on the Live queries subsystem page.",
      questionCount: 9,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/live-queries#live-query-sort",
    },
    {
      surface: "flow" as const,
      slug: "vpp-install",
      label: "VPP install: Flow order",
      description:
        "Seven steps from Install click to installed status. Fleet never holds the app bytes; Apple does. Embedded on the VPP (macOS) subsystem page.",
      questionCount: 7,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/software-install-vpp-mac#vpp-install-sort",
    },
    {
      surface: "flow" as const,
      slug: "managed-google-play",
      label: "Managed Google Play: Install lifecycle order",
      description:
        "Eight steps. Fleet never touches the device; Google is in the middle of every hop. Embedded on the Managed Google Play subsystem page.",
      questionCount: 8,
      surfaceLabel: "Flow order",
      surfaceHref:
        "/subsystems/software-install-managed-google-play#managed-google-play-sort",
    },
    {
      surface: "flow" as const,
      slug: "inhouse-install",
      label: "In-house install: Lifecycle order",
      description:
        "Eight steps. Fleet owns the IPA bytes and serves them directly; the OS validates the enterprise signature before installing. Embedded on the In-house apps subsystem page.",
      questionCount: 8,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/software-install-inhouse#inhouse-install-sort",
    },
    {
      surface: "flow" as const,
      slug: "custom-install",
      label: "Custom installer: Lifecycle order",
      description:
        "Nine steps. Pull-based through fleetd's osquery config poll; no MDM. The three-script pattern runs on the device. Embedded on the Custom installers subsystem page.",
      questionCount: 9,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/software-install-custom#custom-install-sort",
    },
    {
      surface: "flow" as const,
      slug: "day-in-the-life",
      label: "Dependency chain of a Fleet host",
      description:
        "Six steps where each reads what the previous wrote: Orbit → osquery enrollment → host_software → vuln matches → policy_membership → Fleet Desktop. Embedded on the osquery fundamentals subsystem page.",
      questionCount: 6,
      surfaceLabel: "Flow order",
      surfaceHref: "/subsystems/osquery-fundamentals#day-in-the-life-sort",
    },
  ];
  const cron = [
    {
      surface: "cron" as const,
      slug: "which-cron",
      label: "Which cron? adaptive multiple choice",
      description:
        "Match symptoms to the Fleet cron that handles them. Difficulty adapts as you answer. Embedded at the bottom of the Cron schedules page.",
      questionCount: CRON_QUESTIONS.length,
      surfaceLabel: "Which cron?",
      surfaceHref: "/background/cron#cron-quiz",
    },
  ];
  return [
    ...glossary,
    ...arch,
    ...tracer,
    ...tier,
    ...matching,
    ...reviewAI,
    ...values,
    ...flow,
    ...cron,
  ];
}

export const STORAGE_KEYS: Record<SurfaceKind, string> = {
  glossary: "fleet-atlas:quiz-results:glossary",
  architecture: "fleet-atlas:quiz-results:architecture",
  tracer: "fleet-atlas:tracer-quiz-best",
  tier: "fleet-atlas:tier-quiz-best",
  matching: "fleet-atlas:matching-quiz-best",
  "review-ai": "fleet-atlas:review-ai:best",
  values: "fleet-atlas:values-quiz-best",
  flow: "fleet-atlas:sso-saml-sort-quiz-best",
  cron: "fleet-atlas:cron-quiz-best",
};
