"use client";

import { createContext } from "react";

// Shared hover state for ArchGraph. Lives outside the nodes/edges arrays so
// hover state changes don't trigger ReactFlow's internal viewport
// re-measurement (which otherwise strobes visibility across every node).
export type ArchHoverState = {
  hoveredNodeId: string | null;
  hoveredEdgeId: string | null;
  hoveredEdgeSource: string | null;
  hoveredEdgeTarget: string | null;
};

export const ArchHoverCtx = createContext<ArchHoverState>({
  hoveredNodeId: null,
  hoveredEdgeId: null,
  hoveredEdgeSource: null,
  hoveredEdgeTarget: null,
});
