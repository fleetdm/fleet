import { GROUP_META, type NodeGroup } from "./graph-data";

export default function Legend() {
  const groups = Object.keys(GROUP_META) as NodeGroup[];
  return (
    <div className="flex flex-wrap gap-2 text-xs">
      {groups.map((g) => {
        const m = GROUP_META[g];
        return (
          <span
            key={g}
            className={`inline-flex items-center gap-1.5 rounded border ${m.chip} px-2 py-0.5 text-zinc-700 dark:text-zinc-300`}
          >
            <span className={`inline-block h-2 w-2 rounded-sm border ${m.chip}`} />
            {m.label}
          </span>
        );
      })}
    </div>
  );
}
