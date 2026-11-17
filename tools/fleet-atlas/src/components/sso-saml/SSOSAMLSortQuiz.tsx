"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { SSO_SAML_STEPS } from "./steps";

export default function SSOSAMLSortQuiz() {
  return (
    <SortQuiz
      title="SSO SAML login. Admin flow order"
      description='Seven steps from "click Log in with SSO" to minted session. Drag them into the right order, then submit. Tab + Space activates keyboard drag.'
      storageKeyBest="fleet-atlas:sso-saml-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:sso-saml-sort-quiz-in-progress"
      steps={SSO_SAML_STEPS}
    />
  );
}
