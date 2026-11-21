"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useTechnicalHelp } from "@/providers/TechnicalHelp";
import GlossaryTermChip from "./GlossaryTermChip";

// Section headings that are hidden when Technical help mode is off.
// Match is case-insensitive and matches the start of the h3 line.
const TECHNICAL_HELP_HEADINGS = [/^###\s*Technical help/i];
const TECH_HELP_HEADING_RE = /^###\s*Technical help\b/i;

function filterTechnicalHelp(md: string, on: boolean): string {
  if (on) return md;
  const lines = md.split("\n");
  const out: string[] = [];
  let skipping = false;
  for (const line of lines) {
    const isHeader = /^#{1,3}\s/.test(line);
    if (skipping && isHeader) skipping = false;
    if (isHeader && TECHNICAL_HELP_HEADINGS.some((re) => re.test(line))) {
      skipping = true;
      continue;
    }
    if (!skipping) out.push(line);
  }
  return out.join("\n");
}

type Chunk = { kind: "default" | "tech-help"; md: string };

// Split markdown into chunks at top-level h3 boundaries, flagging any chunk
// whose opening heading is "Technical help" so the renderer can wrap it in a
// blue card. Fenced code blocks are respected so a stray ### inside a code
// block doesn't split the section.
function splitIntoChunks(md: string): Chunk[] {
  const lines = md.split("\n");
  const chunks: Chunk[] = [];
  let buf: string[] = [];
  let kind: Chunk["kind"] = "default";
  let inFence = false;
  const flush = () => {
    if (buf.length === 0) return;
    chunks.push({ kind, md: buf.join("\n") });
    buf = [];
  };
  for (const line of lines) {
    if (/^```/.test(line)) inFence = !inFence;
    const isH3 = !inFence && /^###\s/.test(line);
    if (isH3) {
      flush();
      kind = TECH_HELP_HEADING_RE.test(line) ? "tech-help" : "default";
    }
    buf.push(line);
  }
  flush();
  return chunks;
}

// Known top-level directories + files in the fleetdm/fleet repo. If an inline
// `code` string starts with one of these, we render it as a GitHub link.
const FLEET_PATH_ROOTS = [
  "cmd/",
  "server/",
  "ee/",
  "frontend/",
  "docs/",
  "handbook/",
  "orbit/",
  "tools/",
  ".github/",
  "pkg/",
];
const FLEET_PATH_FILES = [
  "Makefile",
  "go.mod",
  "go.sum",
  "package.json",
  ".golangci.yml",
];

function fleetGithubUrl(path: string): string | null {
  // Trim trailing punctuation that commonly ends a sentence (period, colon, comma).
  const trimmed = path.replace(/[.,;:)\]]+$/, "");
  if (FLEET_PATH_FILES.includes(trimmed)) {
    return `https://github.com/fleetdm/fleet/blob/main/${trimmed}`;
  }
  if (FLEET_PATH_ROOTS.some((r) => trimmed.startsWith(r))) {
    return `https://github.com/fleetdm/fleet/blob/main/${trimmed}`;
  }
  return null;
}

function renderMarkdown(md: string, techHelpHeadingVariant: "chip" | "plain") {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => (
          <h1 className="mt-10 mb-4 text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
            {children}
          </h1>
        ),
        h2: ({ children }) => (
          <h2 className="mt-10 mb-3 text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            {children}
          </h2>
        ),
        h3: ({ children }) => {
          const text =
            Array.isArray(children) && typeof children[0] === "string"
              ? children[0]
              : typeof children === "string"
                ? children
                : "";
          const isTechHelp = /^\s*technical help\b/i.test(text);
          if (isTechHelp && techHelpHeadingVariant === "chip") {
            return (
              <h3 className="mt-1 mb-3 flex items-center">
                <span className="inline-flex items-center gap-1 text-sm font-semibold text-sky-800 dark:text-sky-200">
                  💡 {children}
                </span>
              </h3>
            );
          }
          if (isTechHelp) {
            return (
              <h3 className="mt-7 mb-2 flex items-center">
                <span className="inline-flex items-center gap-1 rounded-full border border-sky-300 bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-800 dark:border-sky-700 dark:bg-sky-950/60 dark:text-sky-200">
                  💡 {children}
                </span>
              </h3>
            );
          }
          return (
            <h3 className="mt-7 mb-2 border-b border-zinc-200 pb-1 text-sm font-semibold uppercase tracking-wide text-zinc-700 dark:border-zinc-800 dark:text-zinc-300">
              {children}
            </h3>
          );
        },
        h4: ({ children }) => (
          <h4 className="mt-5 mb-2 text-base font-semibold text-zinc-800 dark:text-zinc-200">
            {children}
          </h4>
        ),
        code: ({ children, className, ...props }) => {
          if (className) {
            return (
              <code className={className} {...props}>
                {children}
              </code>
            );
          }
          const text = typeof children === "string" ? children : "";
          const url = text ? fleetGithubUrl(text) : null;
          if (url) {
            return (
              <a
                href={url}
                target="_blank"
                rel="noopener"
                className="no-underline"
              >
                <code {...props}>{children}</code>
              </a>
            );
          }
          return <code {...props}>{children}</code>;
        },
        a: ({ href, children }) => {
          if (href && href.startsWith("glossary:")) {
            const slug = href.slice("glossary:".length);
            return <GlossaryTermChip slug={slug}>{children}</GlossaryTermChip>;
          }
          return (
            <a
              href={href}
              target={href?.startsWith("http") ? "_blank" : undefined}
              rel={href?.startsWith("http") ? "noopener" : undefined}
            >
              {children}
            </a>
          );
        },
      }}
    >
      {md}
    </ReactMarkdown>
  );
}

export default function Markdown({ children }: { children: string }) {
  const { on } = useTechnicalHelp();
  const content = filterTechnicalHelp(children, on);
  const chunks = splitIntoChunks(content);
  return (
    <div className="prose prose-zinc dark:prose-invert max-w-none prose-code:rounded prose-code:bg-zinc-100 prose-code:px-1 prose-code:py-0.5 prose-code:text-sm prose-code:text-zinc-900 prose-code:before:content-none prose-code:after:content-none dark:prose-code:bg-zinc-800 dark:prose-code:text-zinc-100 prose-pre:bg-zinc-900 prose-pre:text-zinc-100 dark:prose-pre:bg-zinc-950">
      {chunks.map((chunk, i) => {
        if (chunk.kind === "tech-help") {
          return (
            <div
              key={i}
              className="not-prose my-6 rounded border border-sky-200 bg-sky-50/60 p-5 dark:border-sky-800 dark:bg-sky-950/30"
            >
              <div className="prose prose-zinc dark:prose-invert max-w-none prose-code:rounded prose-code:bg-sky-100 prose-code:px-1 prose-code:py-0.5 prose-code:text-sm prose-code:text-sky-900 prose-code:before:content-none prose-code:after:content-none dark:prose-code:bg-sky-900/60 dark:prose-code:text-sky-100">
                {renderMarkdown(chunk.md, "chip")}
              </div>
            </div>
          );
        }
        return <div key={i}>{renderMarkdown(chunk.md, "plain")}</div>;
      })}
    </div>
  );
}
