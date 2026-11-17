"use client";

import { useContext } from "react";
import { EdgeProps, getBezierPath } from "reactflow";
import { useTheme } from "@/components/Theme";
import { ArchHoverCtx } from "./hover-context";

type EdgeData = { tooltip?: string };

export default function TooltipEdge(props: EdgeProps<EdgeData>) {
  const {
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    style,
    markerEnd,
    label,
    labelStyle,
    labelBgStyle,
  } = props;

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const { hoveredEdgeId } = useContext(ArchHoverCtx);
  const { theme } = useTheme();
  const isHovered = hoveredEdgeId === props.id;
  const emphasisColor = theme === "dark" ? "#e4e4e7" : "#27272a";
  const emphasizedStyle = isHovered
    ? { ...style, stroke: emphasisColor, strokeWidth: 2 }
    : style;

  return (
    <g style={{ cursor: "pointer" }}>
      <path
        id={props.id}
        d={edgePath}
        style={emphasizedStyle}
        markerEnd={markerEnd}
        className="react-flow__edge-path"
        fill="none"
      />
      <path
        d={edgePath}
        style={{
          stroke: "transparent",
          strokeWidth: 32,
          fill: "none",
        }}
      />
      {label ? (
        <g transform={`translate(${labelX}, ${labelY})`}>
          <rect
            x={-((String(label).length * 6) / 2) - 4}
            y={-7}
            width={String(label).length * 6 + 8}
            height={14}
            rx={2}
            style={labelBgStyle}
          />
          <text
            textAnchor="middle"
            dominantBaseline="central"
            style={{ ...labelStyle, pointerEvents: "none" }}
          >
            {label}
          </text>
        </g>
      ) : null}
    </g>
  );
}
