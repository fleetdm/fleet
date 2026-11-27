import { TAG_COLORS, type ScenarioTag } from "./scenarios";

export default function TagChip({
  tag,
  active,
  onClick,
  as = "span",
}: {
  tag: ScenarioTag;
  active?: boolean;
  onClick?: () => void;
  as?: "span" | "button";
}) {
  const base = TAG_COLORS[tag];
  const buttonClass = `inline-flex items-center rounded px-2 py-0.5 text-[11px] font-medium cursor-pointer ${base} ${
    active ? "ring-1 ring-zinc-900 dark:ring-zinc-100" : ""
  }`;
  const spanClass = `inline-flex items-center rounded px-2 py-0.5 text-[11px] font-medium ${base} ${
    active ? "ring-1 ring-zinc-900 dark:ring-zinc-100" : ""
  }`;
  if (as === "button") {
    return (
      <button type="button" onClick={onClick} className={buttonClass}>
        {tag}
      </button>
    );
  }
  return <span className={spanClass}>{tag}</span>;
}
