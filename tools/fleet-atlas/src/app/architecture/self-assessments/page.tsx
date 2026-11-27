import { Suspense } from "react";
import MapQuiz from "@/components/arch/MapQuiz";

export default function AssessmentsPage() {
  return (
    <div className="h-full overflow-y-auto">
      <Suspense fallback={null}>
        <MapQuiz />
      </Suspense>
    </div>
  );
}
