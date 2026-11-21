"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { INHOUSE_INSTALL_STEPS } from "./steps";

export default function InhouseInstallSortQuiz() {
  return (
    <SortQuiz
      title="In-house install. Lifecycle order"
      description="Eight steps. Fleet owns the IPA bytes and serves them directly; the OS validates the enterprise signature before installing. Drag into order, then submit."
      storageKeyBest="fleet-atlas:inhouse-install-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:inhouse-install-sort-quiz-in-progress"
      steps={INHOUSE_INSTALL_STEPS}
    />
  );
}
