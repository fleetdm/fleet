import {
  CRONS,
  TOPIC_ORDER,
  cadenceLabel,
  type CronJob,
  type CronTopic,
} from "./jobs";

const ROW_H = 20;
const TOPIC_HEADER_H = 22;
const TOP_PAD = 30;
const LEFT_PAD = 240;
const RIGHT_PAD = 70;
const WIDTH = 980;

const TOPIC_FILL: Record<CronTopic, string> = {
  Housekeeping: "fill-zinc-500 dark:fill-zinc-400",
  "Automations & integrations": "fill-sky-500 dark:fill-sky-400",
  "Batch activities": "fill-indigo-500 dark:fill-indigo-400",
  Vulnerabilities: "fill-fuchsia-500 dark:fill-fuchsia-400",
  "Software catalog (FMAs & VPP)": "fill-emerald-500 dark:fill-emerald-400",
  "MDM: Apple": "fill-pink-500 dark:fill-pink-400",
  "MDM: Microsoft": "fill-cyan-500 dark:fill-cyan-400",
  "MDM: Android": "fill-lime-500 dark:fill-lime-400",
  "Calendar & Workspace": "fill-amber-500 dark:fill-amber-400",
  "Reports (query results)": "fill-violet-500 dark:fill-violet-400",
  "One-shot migrations": "fill-rose-500 dark:fill-rose-400",
};

export default function Timeline() {
  const trackWidth = WIDTH - LEFT_PAD - RIGHT_PAD;

  const grouped: { topic: CronTopic; items: CronJob[] }[] = TOPIC_ORDER.map(
    (t) => ({
      topic: t,
      items: CRONS.filter((c) => c.topic === t).sort(
        (a, b) => a.intervalMinutes - b.intervalMinutes,
      ),
    }),
  ).filter((g) => g.items.length > 0);

  const rows: Array<
    | { kind: "header"; topic: CronTopic }
    | { kind: "cron"; job: CronJob; topic: CronTopic }
  > = [];
  for (const g of grouped) {
    rows.push({ kind: "header", topic: g.topic });
    for (const job of g.items) rows.push({ kind: "cron", job, topic: g.topic });
  }

  const height =
    TOP_PAD +
    rows.reduce((h, r) => h + (r.kind === "header" ? TOPIC_HEADER_H : ROW_H), 0) +
    10;

  let y = TOP_PAD;
  const positions: Array<{ y: number; row: (typeof rows)[number] }> = [];
  for (const r of rows) {
    positions.push({ y, row: r });
    y += r.kind === "header" ? TOPIC_HEADER_H : ROW_H;
  }

  return (
    <div className="overflow-x-auto rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
      <svg width={WIDTH} height={height} className="block">
        {Array.from({ length: 25 }).map((_, h) => {
          const x = LEFT_PAD + (h / 24) * trackWidth;
          return (
            <g key={h}>
              <line
                x1={x}
                y1={TOP_PAD - 10}
                x2={x}
                y2={height - 6}
                stroke={h % 6 === 0 ? "#71717a" : "#3f3f46"}
                strokeOpacity={h % 6 === 0 ? 0.45 : 0.15}
              />
              {h % 6 === 0 ? (
                <text
                  x={x}
                  y={TOP_PAD - 14}
                  textAnchor="middle"
                  fontSize={10}
                  className="fill-zinc-600 dark:fill-zinc-400"
                >
                  {String(h).padStart(2, "0")}:00
                </text>
              ) : null}
            </g>
          );
        })}

        {positions.map(({ y, row }, idx) => {
          if (row.kind === "header") {
            return (
              <g key={`h-${idx}`}>
                <line
                  x1={0}
                  y1={y + TOPIC_HEADER_H - 2}
                  x2={WIDTH}
                  y2={y + TOPIC_HEADER_H - 2}
                  stroke="#d4d4d8"
                  strokeOpacity={0.3}
                />
                <text
                  x={10}
                  y={y + TOPIC_HEADER_H - 6}
                  fontSize={11}
                  className="fill-zinc-700 dark:fill-zinc-200 font-semibold"
                  style={{ textTransform: "uppercase", letterSpacing: 0.4 }}
                >
                  {row.topic}
                </text>
              </g>
            );
          }
          const c = row.job;
          const cy = y + ROW_H / 2;
          const stepsPerDay = Math.max(
            1,
            Math.round((60 * 24) / c.intervalMinutes),
          );
          const stepPx = trackWidth / stepsPerDay;
          const useBar = stepPx < 4;
          return (
            <g key={`c-${c.name}`}>
              <text
                x={LEFT_PAD - 10}
                y={cy + 4}
                textAnchor="end"
                fontSize={10}
                className="fill-zinc-700 dark:fill-zinc-300 font-mono"
              >
                {c.name}
              </text>
              {useBar ? (
                <rect
                  x={LEFT_PAD}
                  y={cy - 3}
                  width={trackWidth}
                  height={6}
                  className={`${TOPIC_FILL[row.topic]} opacity-40`}
                  rx={3}
                />
              ) : (
                Array.from({ length: stepsPerDay }).map((_, k) => (
                  <circle
                    key={k}
                    cx={LEFT_PAD + k * stepPx}
                    cy={cy}
                    r={2.5}
                    className={TOPIC_FILL[row.topic]}
                  />
                ))
              )}
              <text
                x={LEFT_PAD + trackWidth + 6}
                y={cy + 4}
                fontSize={9}
                className="fill-zinc-500"
              >
                {cadenceLabel(c.intervalMinutes)}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}
