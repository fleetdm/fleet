// Single source of truth for which subsystems have embedded quizzes on their
// detail page. Both /subsystems (the index. Shows the Quiz chip) and
// /subsystems/[slug] (the detail. Renders the components) read from this map.
// Add a new embedded quiz = add one entry here; everything downstream updates.

import type { ComponentType } from "react";
import SSOSAMLSortQuiz from "@/components/sso-saml/SSOSAMLSortQuiz";
import MacDEPSortQuiz from "@/components/mac-dep/MacDEPSortQuiz";
import SetupExperienceSortQuiz from "@/components/setup-experience-flow/SetupExperienceSortQuiz";
import LiveQuerySortQuiz from "@/components/live-query-flow/LiveQuerySortQuiz";
import VPPInstallSortQuiz from "@/components/vpp-install-flow/VPPInstallSortQuiz";
import ManagedGooglePlaySortQuiz from "@/components/managed-google-play-flow/ManagedGooglePlaySortQuiz";
import InhouseInstallSortQuiz from "@/components/inhouse-install-flow/InhouseInstallSortQuiz";
import CustomInstallSortQuiz from "@/components/custom-install-flow/CustomInstallSortQuiz";
import DayInTheLifeSortQuiz from "@/components/day-in-the-life-flow/DayInTheLifeSortQuiz";
import WebhookTypesMatchingQuiz from "@/components/webhook-types-matching/WebhookTypesMatchingQuiz";

type EmbeddedQuiz = {
  // Anchor id on the detail page so self-assessments links + in-page refs can
  // scroll to this section.
  anchor: string;
  Component: ComponentType;
};

export const SUBSYSTEM_EMBEDDED_QUIZZES: Record<string, EmbeddedQuiz[]> = {
  "sso-scim": [{ anchor: "sso-saml-sort", Component: SSOSAMLSortQuiz }],
  "apple-mdm": [
    { anchor: "mac-dep-sort", Component: MacDEPSortQuiz },
    { anchor: "setup-experience-sort", Component: SetupExperienceSortQuiz },
  ],
  "live-queries": [{ anchor: "live-query-sort", Component: LiveQuerySortQuiz }],
  "software-install-vpp-mac": [
    { anchor: "vpp-install-sort", Component: VPPInstallSortQuiz },
  ],
  "software-install-managed-google-play": [
    {
      anchor: "managed-google-play-sort",
      Component: ManagedGooglePlaySortQuiz,
    },
  ],
  "software-install-inhouse": [
    { anchor: "inhouse-install-sort", Component: InhouseInstallSortQuiz },
  ],
  "software-install-custom": [
    { anchor: "custom-install-sort", Component: CustomInstallSortQuiz },
  ],
  "osquery-fundamentals": [
    { anchor: "day-in-the-life-sort", Component: DayInTheLifeSortQuiz },
  ],
  "webhooks-integrations": [
    { anchor: "webhook-types-matching", Component: WebhookTypesMatchingQuiz },
  ],
};

export function subsystemHasEmbeddedQuiz(slug: string): boolean {
  return (SUBSYSTEM_EMBEDDED_QUIZZES[slug]?.length ?? 0) > 0;
}

export function embeddedQuizzesFor(slug: string): EmbeddedQuiz[] {
  return SUBSYSTEM_EMBEDDED_QUIZZES[slug] ?? [];
}
