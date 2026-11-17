import Link from "next/link";
import { notFound } from "next/navigation";
import CodeSnippet from "@/components/CodeSnippet";
import { TechnicalHelp } from "@/providers/TechnicalHelp";
import Markdown from "@/components/Markdown";
import { getTracer } from "@/components/tracer/tracers";
import TracerStepQuiz from "@/components/tracer/TracerStepQuiz";

export const instant = false;

const LAYER_COLORS: Record<string, string> = {
  Router:
    "border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950 dark:text-sky-200",
  Decoder:
    "border-cyan-300 bg-cyan-50 text-cyan-800 dark:border-cyan-700 dark:bg-cyan-950 dark:text-cyan-200",
  Endpoint:
    "border-teal-300 bg-teal-50 text-teal-800 dark:border-teal-700 dark:bg-teal-950 dark:text-teal-200",
  Service:
    "border-indigo-300 bg-indigo-50 text-indigo-800 dark:border-indigo-700 dark:bg-indigo-950 dark:text-indigo-200",
  Authz:
    "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-200",
  Datastore:
    "border-violet-300 bg-violet-50 text-violet-800 dark:border-violet-700 dark:bg-violet-950 dark:text-violet-200",
  MySQL:
    "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950 dark:text-emerald-200",
  Response:
    "border-rose-300 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950 dark:text-rose-200",
};

export default async function TracerDetail({
  params,
}: PageProps<"/tracer/[slug]">) {
  const { slug } = await params;
  const tracer = getTracer(slug);
  if (!tracer) notFound();

  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link href="/tracer" className="hover:text-zinc-900 dark:hover:text-zinc-200">
          ← Tracers
        </Link>
      </nav>

      <header className="mb-8 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="flex items-center gap-3">
          <span className="rounded bg-zinc-200 px-2 py-0.5 font-mono text-xs text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200">
            {tracer.method}
          </span>
          <span className="font-mono text-sm text-zinc-700 dark:text-zinc-300">
            {tracer.path}
          </span>
        </div>
        <h1 className="mt-2 text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          {tracer.title}
        </h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          {tracer.summary}
        </p>
      </header>

      {tracer.stub ? (
        <div className="rounded border border-dashed border-zinc-300 bg-zinc-50 p-6 text-sm text-zinc-600 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
          Trace coming soon. For now, read the summary and come back later.
        </div>
      ) : (
        <>
          <section className="mb-10">
            <Markdown>{tracer.overview}</Markdown>
          </section>

          <ol className="space-y-8">
            {tracer.steps.map((step, i) => (
              <li
                key={`${step.layer}-${i}`}
                className="rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
              >
                <div className="mb-3 flex items-center gap-3">
                  <span className="font-mono text-xs text-zinc-500">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span
                    className={`rounded border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${
                      LAYER_COLORS[step.layer] ?? ""
                    }`}
                  >
                    {step.layer}
                  </span>
                  <h2 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">
                    {step.title}
                  </h2>
                </div>

                <Markdown>{step.narrative}</Markdown>

                {step.snippet ? (
                  <CodeSnippet
                    path={step.snippet.path}
                    start={step.snippet.start}
                    end={step.snippet.end}
                    caption={step.snippet.caption}
                  />
                ) : null}

                {step.technicalHelp ? (
                  <TechnicalHelp
                    technical={step.technicalHelp.technical}
                    analogies={step.technicalHelp.analogies}
                  />
                ) : null}
              </li>
            ))}
          </ol>

          {tracer.takeaways ? (
            <section className="mt-10 rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
              <Markdown>{tracer.takeaways}</Markdown>
            </section>
          ) : null}

          <TracerStepQuiz
            slug={tracer.slug}
            method={tracer.method}
            path={tracer.path}
            steps={tracer.steps}
          />
        </>
      )}
    </div>
  );
}
