"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { MAC_DEP_STEPS } from "./steps";

export default function MacDEPSortQuiz() {
  return (
    <SortQuiz
      title="Mac DEP enrollment. Flow order"
      description="Eight steps from ABM assignment to setup assistant. Drag them into the right order, then submit. Tab + Space activates keyboard drag."
      storageKeyBest="fleet-atlas:mac-dep-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:mac-dep-sort-quiz-in-progress"
      steps={MAC_DEP_STEPS}
    />
  );
}
