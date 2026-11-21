"use client";

import SortQuiz from "@/components/flow-sorts/SortQuiz";
import { LIVE_QUERY_STEPS } from "./steps";

export default function LiveQuerySortQuiz() {
  return (
    <SortQuiz
      title="Live query. Flow order"
      description="Seven steps from the admin's Run click to rows streaming into the browser. Pull-based + Redis pubsub are the non-obvious bits. Drag into order, then submit."
      storageKeyBest="fleet-atlas:live-query-sort-quiz-best"
      storageKeyInProgress="fleet-atlas:live-query-sort-quiz-in-progress"
      steps={LIVE_QUERY_STEPS}
    />
  );
}
