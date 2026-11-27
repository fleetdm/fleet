import Link from "next/link";
import { notFound } from "next/navigation";
import { getScenario } from "@/components/scenarios/scenarios";
import ScenarioBody from "@/components/scenarios/ScenarioBody";
import TagChip from "@/components/scenarios/TagChip";
import { getCustomScenario } from "@/lib/custom-scenarios";

export const instant = false;

export default async function ScenarioDetail({
  params,
}: PageProps<"/scenarios/[slug]">) {
  const { slug } = await params;
  const sc = getScenario(slug) ?? getCustomScenario(slug);
  if (!sc) notFound();

  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link
          href="/scenarios"
          className="hover:text-zinc-900 dark:hover:text-zinc-200"
        >
          ← Scenarios
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
            {sc.label}
          </h1>
          <p className="mt-2 text-zinc-600 dark:text-zinc-400">{sc.summary}</p>
          {sc.tags.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-1">
              {sc.tags.map((t) => (
                <TagChip key={t} tag={t} />
              ))}
            </div>
          ) : null}
        </div>
      </header>

      <ScenarioBody scenario={sc} />
    </div>
  );
}
