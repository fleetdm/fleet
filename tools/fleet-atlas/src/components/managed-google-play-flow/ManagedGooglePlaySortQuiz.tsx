"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { MANAGED_GOOGLE_PLAY_STEPS } from "./steps";

export default function ManagedGooglePlaySortQuiz() {
  return (
    <SortQuiz
      title="Managed Google Play. Install lifecycle order"
      description="Eight steps. Fleet never touches the device; Google is in the middle of every hop. Drag into the right order, then submit."
      storageKeyBest="fleet-atlas:managed-google-play-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:managed-google-play-sort-quiz-in-progress"
      steps={MANAGED_GOOGLE_PLAY_STEPS}
    />
  );
}
