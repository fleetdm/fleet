"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { CUSTOM_INSTALL_STEPS } from "./steps";

export default function CustomInstallSortQuiz() {
  return (
    <SortQuiz
      title="Custom installer. Lifecycle order"
      description="Nine steps. Pull-based delivery through fleetd's osquery config poll. No MDM. The three-script pattern runs on the device. Drag into order, then submit."
      storageKeyBest="fleet-atlas:custom-install-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:custom-install-sort-quiz-in-progress"
      steps={CUSTOM_INSTALL_STEPS}
    />
  );
}
