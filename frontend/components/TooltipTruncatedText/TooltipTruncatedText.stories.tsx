import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import TooltipTruncatedText from ".";
import DataSet from "../DataSet";

import "../../index.scss";

const meta: Meta<typeof TooltipTruncatedText> = {
  component: TooltipTruncatedText,
  title: "Components/TooltipTruncatedText",
  args: {
    value:
      "This is an example of a very long text that will be truncated in the display area.",
  },
  decorators: [
    (Story) => (
      <div style={{ maxWidth: "200px", padding: "150px", overflow: "visible" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof TooltipTruncatedText>;

// Drag the `containerWidth` control in the Storybook addons panel to resize the
// container. The tooltip should only appear when the container is narrow enough
// that the text actually overflows.
export const Default: StoryObj<
  React.ComponentProps<typeof TooltipTruncatedText> & { containerWidth: number }
> = {
  args: {
    containerWidth: 200,
    value: "Resize the container to show tooltip only shows on truncation",
  },
  argTypes: {
    containerWidth: {
      control: { type: "range", min: 50, max: 900, step: 10 },
      description:
        "Width of the wrapping container in px. Tooltip only shows when the text is truncated.",
    },
  },
  render: ({ containerWidth, ...args }) => (
    <div
      style={{
        width: containerWidth,
        padding: "8px",
        border: "1px dashed #c5c7d1",
      }}
    >
      <TooltipTruncatedText {...args} />
    </div>
  ),
};

export const UsedInsideDataSet: Story = {
  decorators: [
    (Story) => (
      <DataSet
        className="my-dataset"
        title="Example Title"
        value={<Story />}
        orientation="horizontal"
      />
    ),
  ],
};
