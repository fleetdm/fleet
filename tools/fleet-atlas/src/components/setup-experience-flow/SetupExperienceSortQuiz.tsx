"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { SETUP_EXPERIENCE_STEPS } from "./steps";

export default function SetupExperienceSortQuiz() {
  return (
    <SortQuiz
      title="Setup experience install. Flow order"
      description="Eight steps. Fleet pre-installs apps while the Mac is paused at 'awaiting configuration,' then releases via DDM. Drag into order, then submit."
      storageKeyBest="fleet-atlas:setup-experience-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:setup-experience-sort-quiz-in-progress"
      steps={SETUP_EXPERIENCE_STEPS}
    />
  );
}
