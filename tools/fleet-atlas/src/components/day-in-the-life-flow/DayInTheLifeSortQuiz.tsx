"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { DAY_IN_THE_LIFE_STEPS } from "./steps";

export default function DayInTheLifeSortQuiz() {
  return (
    <SortQuiz
      title="Dependency chain of a Fleet host"
      description="Six steps where each one reads what the previous one wrote: Orbit → osquery enrollment → host_software populated → vuln matches → policy_membership → Fleet Desktop surfaces failures. Not a clock timeline. A data dependency chain."
      storageKeyBest="fleet-atlas:day-in-the-life-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:day-in-the-life-sort-quiz-in-progress"
      steps={DAY_IN_THE_LIFE_STEPS}
    />
  );
}
