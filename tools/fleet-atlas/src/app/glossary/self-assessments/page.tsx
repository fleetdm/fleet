import { Suspense } from "react";
import Quiz from "@/components/glossary/Quiz";

export default function SelfAssessmentsPage() {
  return (
    <Suspense fallback={null}>
      <Quiz />
    </Suspense>
  );
}
