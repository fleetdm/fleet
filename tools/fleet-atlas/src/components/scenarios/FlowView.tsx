"use client";

import { useMemo } from "react";
import ReactFlow, {
  Background,
  Edge,
  Handle,
  MarkerType,
  Node,
  NodeProps,
  Position,
} from "reactflow";
import "reactflow/dist/style.css";
import { useTheme } from "@/providers/Theme";
import { ACTOR_COLORS, type Actor, type Step } from "./scenarios";

const COL_W = 280;
const COL_GAP = 40;
const ROW_H = 150;
const HEADER_H = 36;

type NodeData = Step & { laneColor: string };

function StepNode({ data }: NodeProps<NodeData>) {
  return (
    <div
      className="rounded border border-zinc-300 bg-white p-3 shadow-sm dark:border-zinc-700 dark:bg-zinc-950"
      style={{ width: COL_W }}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-zinc-500 !border-none"
      />
      <Handle
        type="target"
        position={Position.Left}
        id="l"
        className="!bg-zinc-500 !border-none"
      />
      <div className="mb-1 flex items-center gap-2">
        <span className="font-mono text-[11px] text-zinc-500">{data.t}</span>
        <span
          className={`rounded px-2 py-0.5 text-[11px] font-semibold ${data.laneColor}`}
        >
          {data.actor}
        </span>
      </div>
      <div className="text-sm text-zinc-900 dark:text-zinc-100">
        {data.action}
      </div>
      {data.detail ? (
        <div className="mt-1 text-xs text-zinc-500">{data.detail}</div>
      ) : null}
      {data.file ? (
        <div className="mt-1 font-mono text-[11px] text-zinc-400">
          {data.file}
        </div>
      ) : null}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-zinc-500 !border-none"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="r"
        className="!bg-zinc-500 !border-none"
      />
    </div>
  );
}

function LaneHeaderNode({ data }: NodeProps<{ actor: Actor; color: string }>) {
  return (
    <div
      className={`rounded px-3 py-1 text-center text-xs font-semibold ${data.color}`}
      style={{ width: COL_W }}
    >
      {data.actor}
    </div>
  );
}

const nodeTypes = { step: StepNode, laneHeader: LaneHeaderNode };

export default function FlowView({ steps }: { steps: Step[] }) {
  const { theme } = useTheme();
  const isDark = theme === "dark";
  const edgeStroke = isDark ? "#71717a" : "#a1a1aa";
  const bgDot = isDark ? "#27272a" : "#e4e4e7";
  const laneBg = isDark ? "#18181b" : "#fafafa";
  const laneBorder = isDark ? "#27272a" : "#e4e4e7";

  const { nodes, edges, lanes, width, height } = useMemo(() => {
    const laneOrder: Actor[] = [];
    for (const s of steps) {
      if (!laneOrder.includes(s.actor)) laneOrder.push(s.actor);
    }
    const laneX: Record<string, number> = {};
    laneOrder.forEach((a, i) => {
      laneX[a] = i * (COL_W + COL_GAP);
    });

    const headerNodes: Node[] = laneOrder.map((actor) => ({
      id: `lane-${actor}`,
      type: "laneHeader",
      position: { x: laneX[actor], y: 0 },
      data: { actor, color: ACTOR_COLORS[actor] },
      draggable: false,
      selectable: false,
    }));

    const stepNodes: Node<NodeData>[] = steps.map((s, i) => ({
      id: String(i),
      type: "step",
      position: {
        x: laneX[s.actor],
        y: HEADER_H + 20 + i * ROW_H,
      },
      data: { ...s, laneColor: ACTOR_COLORS[s.actor] },
    }));

    const stepEdges: Edge[] = steps.slice(1).map((_, i) => {
      const fromActor = steps[i].actor;
      const toActor = steps[i + 1].actor;
      const sameLane = fromActor === toActor;
      return {
        id: `e-${i}`,
        source: String(i),
        target: String(i + 1),
        sourceHandle: sameLane ? undefined : laneX[toActor] > laneX[fromActor] ? "r" : undefined,
        targetHandle: sameLane ? undefined : laneX[toActor] > laneX[fromActor] ? "l" : "l",
        type: "smoothstep",
        animated: true,
        style: { stroke: edgeStroke, strokeWidth: 1.5 },
        markerEnd: { type: MarkerType.ArrowClosed, color: edgeStroke },
      };
    });

    const width = laneOrder.length * (COL_W + COL_GAP);
    const height = HEADER_H + 20 + steps.length * ROW_H + 60;

    return {
      nodes: [...headerNodes, ...stepNodes],
      edges: stepEdges,
      lanes: laneOrder.map((a) => ({ actor: a, x: laneX[a] })),
      width,
      height,
    };
  }, [steps, edgeStroke]);

  return (
    <div className="relative h-full w-full bg-white dark:bg-zinc-950">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        minZoom={0.3}
        proOptions={{ hideAttribution: true }}
      >
        <Background color={bgDot} gap={24} />
        {/* Lane background strips drawn via SVG inside Background's parent */}
        <svg
          width={width}
          height={height}
          className="pointer-events-none absolute left-0 top-0"
          style={{ zIndex: 0 }}
        >
          {lanes.map((l) => (
            <rect
              key={l.actor}
              x={l.x - 10}
              y={HEADER_H + 10}
              width={COL_W + 20}
              height={height - HEADER_H - 20}
              fill={laneBg}
              stroke={laneBorder}
              strokeDasharray="4 4"
              rx={6}
            />
          ))}
        </svg>
      </ReactFlow>
    </div>
  );
}
