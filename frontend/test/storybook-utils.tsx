import React from "react";

/** Storybook decorator factory that wraps a story in a fixed-width bordered
 * frame with generous vertical padding. The padding leaves room for tooltips
 * (and other above/below affordances) that would otherwise be clipped by the
 * Storybook canvas. Use one frame per story to avoid nested wrappers — do not
 * apply both a meta-level and a story-level decorator. */
const withFrame = (width: number) => (Story: React.ComponentType) => (
  <div style={{ width, border: "1px dashed #ccc", padding: "80px 8px" }}>
    <Story />
  </div>
);

export default withFrame;
