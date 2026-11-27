import { editorLink } from "@/lib/config";
import { readFleetLines } from "@/lib/fleet-repo";

type Props = {
  path: string;
  start: number;
  end: number;
  caption?: string;
  highlight?: number[];
};

export default async function CodeSnippet({
  path: filePath,
  start,
  end,
  caption,
  highlight = [],
}: Props) {
  let content = "";
  let error: string | null = null;
  try {
    const result = await readFleetLines(filePath, start, end);
    content = result.content;
  } catch (e) {
    error = e instanceof Error ? e.message : String(e);
  }

  const lines = content.split("\n");

  return (
    <figure className="my-4 overflow-hidden rounded-lg border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950">
      <figcaption className="flex items-center justify-between border-b border-zinc-200 bg-white px-3 py-2 text-xs text-zinc-600 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
        <a
          href={editorLink(filePath, start)}
          className="font-mono text-zinc-800 hover:text-black dark:text-zinc-300 dark:hover:text-white"
          title="Open in editor"
        >
          {filePath}:{start}
          {end > start ? `-${end}` : ""}
        </a>
        {caption ? <span className="italic">{caption}</span> : null}
      </figcaption>
      {error ? (
        <pre className="p-3 text-xs text-rose-600 dark:text-rose-400">
          {error}
        </pre>
      ) : (
        <pre className="overflow-x-auto p-3 text-xs leading-relaxed text-zinc-900 dark:text-zinc-100">
          <code>
            {lines.map((line, i) => {
              const lineNum = start + i;
              const isHl = highlight.includes(lineNum);
              return (
                <div
                  key={lineNum}
                  className={
                    isHl
                      ? "-mx-3 bg-amber-100 px-3 dark:bg-amber-900/30"
                      : ""
                  }
                >
                  <span className="mr-3 inline-block w-10 select-none text-right text-zinc-400 dark:text-zinc-600">
                    {lineNum}
                  </span>
                  {line || " "}
                </div>
              );
            })}
          </code>
        </pre>
      )}
    </figure>
  );
}
