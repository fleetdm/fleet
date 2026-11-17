"use client";

import { useRouter } from "next/navigation";
import { memo, useContext, useEffect, useMemo, useState } from "react";
import ReactFlow, {
  Background,
  Controls,
  Edge,
  MarkerType,
  Node,
  NodeProps,
  Handle,
  Position,
  ReactFlowProvider,
  useReactFlow,
} from "reactflow";
import "reactflow/dist/style.css";
import { useTheme } from "@/components/Theme";
import { GROUP_META, type ArchEdge, type ArchNode } from "./graph-data";
import TooltipEdge from "./TooltipEdge";
import { ArchHoverCtx, type ArchHoverState } from "./hover-context";
import { useFEAnalogue } from "@/components/FEAnalogue";

function GraphNodeBase({ data }: NodeProps<ArchNode>) {
  const meta = GROUP_META[data.group];
  const { hoveredNodeId, hoveredEdgeSource, hoveredEdgeTarget } =
    useContext(ArchHoverCtx);
  const { theme } = useTheme();
  const isHovered =
    hoveredNodeId === data.id ||
    hoveredEdgeSource === data.id ||
    hoveredEdgeTarget === data.id;
  const ringColor = theme === "dark" ? "#e4e4e7" : "#27272a";
  return (
    <div
      className={`relative rounded border ${meta.chip} px-3 py-2 shadow-md`}
      style={{
        width: 180,
        boxShadow: isHovered ? `0 0 0 1px ${ringColor}` : undefined,
      }}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-zinc-500 !border-none"
      />
      <div className="flex items-center justify-between gap-2 pr-5">
        <span className="text-[10px] uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
          {meta.label}
        </span>
        {data.premium ? (
          <span className="rounded border border-amber-400 bg-amber-100 px-1 text-[9px] font-bold uppercase tracking-wide text-amber-800 dark:border-amber-700 dark:bg-amber-950/60 dark:text-amber-200">
            Premium
          </span>
        ) : null}
      </div>
      <div className="mt-0.5 pr-5 text-sm font-medium text-zinc-900 dark:text-zinc-100">
        {data.label}
      </div>
      <span
        className="absolute right-1.5 top-1.5 flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-zinc-400 bg-white/70 text-[10px] font-bold text-zinc-600 dark:border-zinc-500 dark:bg-zinc-900/70 dark:text-zinc-300"
        onClick={(e) => e.stopPropagation()}
      >
        ?
      </span>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-zinc-500 !border-none"
      />
    </div>
  );
}

const GraphNode = memo(GraphNodeBase);
GraphNode.displayName = "GraphNode";

const nodeTypes = { arch: GraphNode };
const edgeTypes = { tooltip: TooltipEdge };

type Props = { nodes: ArchNode[]; edges: ArchEdge[] };
type Hover =
  | { kind: "node"; id: string; text: string; x: number; y: number }
  | {
      kind: "edge";
      id: string;
      source: string;
      target: string;
      text: string;
      techHelp?: string;
      x: number;
      y: number;
    }
  | null;

export default function ArchGraph(props: Props) {
  return (
    <ReactFlowProvider>
      <ArchGraphInner {...props} />
    </ReactFlowProvider>
  );
}

function ArchGraphInner({ nodes: inNodes, edges: inEdges }: Props) {
  const router = useRouter();
  const rf = useReactFlow();

  // One-shot fit on mount only.
  useEffect(() => {
    rf.fitView({ padding: 0.08, maxZoom: 1 });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const { theme } = useTheme();
  const isDark = theme === "dark";
  const edgeStroke = isDark ? "#71717a" : "#a1a1aa";
  const edgeDashed = isDark ? "#52525b" : "#d4d4d8";
  const labelFill = isDark ? "#a1a1aa" : "#52525b";
  const labelBg = isDark ? "#18181b" : "#fafafa";
  const bgDot = isDark ? "#27272a" : "#e4e4e7";

  const [hover, setHover] = useState<Hover>(null);

  // Nodes array depends ONLY on inNodes. Not on hover. ReactFlow receives
  // a stable prop reference across hover state changes, so no viewport
  // re-measure, no visibility strobing.
  const nodes = useMemo<Node<ArchNode>[]>(
    () =>
      inNodes.map((n) => ({
        id: n.id,
        type: "arch",
        position: { x: n.x, y: n.y },
        data: n,
      })),
    [inNodes],
  );

  // Edges array also depends only on static input + theme. Not on hover.
  const edges = useMemo<Edge[]>(
    () =>
      inEdges.map((e, i) => {
        const id = `${e.from}-${e.to}-${i}`;
        const fromLabel = inNodes.find((n) => n.id === e.from)?.label ?? e.from;
        const toLabel = inNodes.find((n) => n.id === e.to)?.label ?? e.to;
        const tooltip =
          e.tooltip ??
          `${fromLabel} → ${toLabel}${e.label ? `: ${e.label}` : ""}`;
        const stroke = e.dashed ? edgeDashed : edgeStroke;
        return {
          id,
          source: e.from,
          target: e.to,
          type: "tooltip",
          label: e.label,
          animated: !e.dashed,
          data: { tooltip, techHelp: e.techHelp },
          style: {
            stroke,
            strokeWidth: 1,
            strokeDasharray: e.dashed ? "4 4" : undefined,
          },
          labelStyle: { fill: labelFill, fontSize: 10 },
          labelBgStyle: { fill: labelBg },
          markerEnd: { type: MarkerType.ArrowClosed, color: stroke },
        };
      }),
    [inNodes, inEdges, edgeStroke, edgeDashed, labelFill, labelBg],
  );

  const hoverCtxValue = useMemo<ArchHoverState>(
    () => ({
      hoveredNodeId: hover?.kind === "node" ? hover.id : null,
      hoveredEdgeId: hover?.kind === "edge" ? hover.id : null,
      hoveredEdgeSource: hover?.kind === "edge" ? hover.source : null,
      hoveredEdgeTarget: hover?.kind === "edge" ? hover.target : null,
    }),
    [hover],
  );

  return (
    <ArchHoverCtx.Provider value={hoverCtxValue}>
      <div className="relative h-full w-full rounded border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          onNodeClick={(_, node) => router.push(`/arch/${node.id}`)}
          onNodeMouseEnter={(event, node) =>
            setHover({
              kind: "node",
              id: node.id,
              text: (node.data as ArchNode).blurb,
              x: event.clientX,
              y: event.clientY,
            })
          }
          onNodeMouseLeave={() => setHover(null)}
          onEdgeMouseEnter={(event, edge) => {
            const data = edge.data as
              | { tooltip?: string; techHelp?: string }
              | undefined;
            setHover({
              kind: "edge",
              id: edge.id,
              source: edge.source,
              target: edge.target,
              text: data?.tooltip ?? "",
              techHelp: data?.techHelp,
              x: event.clientX,
              y: event.clientY,
            });
          }}
          onEdgeMouseLeave={() => setHover(null)}
          defaultEdgeOptions={{ type: "tooltip" }}
          proOptions={{ hideAttribution: true }}
        >
          <Background color={bgDot} gap={24} />
          <Controls showInteractive={false} />
        </ReactFlow>
        {hover && hover.text ? (
          <HoverTooltip
            x={hover.x}
            y={hover.y}
            text={hover.text}
            techHelp={hover.kind === "edge" ? hover.techHelp : undefined}
          />
        ) : null}
      </div>
    </ArchHoverCtx.Provider>
  );
}

function HoverTooltip({
  x,
  y,
  text,
  techHelp,
}: {
  x: number;
  y: number;
  text: string;
  techHelp?: string;
}) {
  const { on: techHelpOn } = useFEAnalogue();
  const showTechHelp = techHelpOn && techHelp;
  return (
    <div
      className="pointer-events-none fixed z-50 max-w-sm rounded border border-zinc-200 bg-white px-2.5 py-1.5 text-xs leading-snug text-zinc-900 shadow-lg dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
      style={{ left: x + 14, top: y + 14 }}
    >
      <div>{text}</div>
      {showTechHelp ? (
        <div className="mt-2 border-t border-zinc-200 pt-2 dark:border-zinc-700">
          <div className="mb-0.5 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide text-sky-700 dark:text-sky-400">
            <span aria-hidden>💡</span>
            Technical help
          </div>
          <div className="text-zinc-700 dark:text-zinc-300">
            {renderTechHelp(techHelp!)}
          </div>
        </div>
      ) : null}
    </div>
  );
}

// Render techHelp with simple markdown-ish support: lines starting with "- "
// become bullet-list items; blank lines break paragraphs. Keeps authoring in
// plain strings while allowing enumerated content to actually read as a list.
function renderTechHelp(text: string): React.ReactNode {
  const lines = text.split("\n");
  const blocks: React.ReactNode[] = [];
  let paragraphLines: string[] = [];
  let bulletLines: string[] = [];

  const flushParagraph = () => {
    if (paragraphLines.length === 0) return;
    blocks.push(
      <p key={`p-${blocks.length}`} className="mt-1 first:mt-0">
        {paragraphLines.join(" ")}
      </p>,
    );
    paragraphLines = [];
  };
  const flushBullets = () => {
    if (bulletLines.length === 0) return;
    blocks.push(
      <ul key={`ul-${blocks.length}`} className="mt-1 list-disc space-y-0.5 pl-4 first:mt-0">
        {bulletLines.map((b, i) => (
          <li key={i}>{b}</li>
        ))}
      </ul>,
    );
    bulletLines = [];
  };

  for (const raw of lines) {
    const line = raw.trimEnd();
    if (line.startsWith("- ")) {
      flushParagraph();
      bulletLines.push(line.slice(2).trim());
    } else if (line.trim() === "") {
      flushParagraph();
      flushBullets();
    } else {
      flushBullets();
      paragraphLines.push(line.trim());
    }
  }
  flushParagraph();
  flushBullets();
  return blocks;
}
