"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { VPP_INSTALL_STEPS } from "./steps";

export default function VPPInstallSortQuiz() {
  return (
    <SortQuiz
      title="VPP install. Flow order"
      description="Seven steps from Install click to installed status. Fleet never holds the app bytes; Apple does. Drag into order, then submit."
      storageKeyBest="fleet-atlas:vpp-install-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:vpp-install-sort-quiz-in-progress"
      steps={VPP_INSTALL_STEPS}
    />
  );
}
