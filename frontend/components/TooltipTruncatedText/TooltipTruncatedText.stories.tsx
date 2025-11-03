import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import TooltipTruncatedText from ".";
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
      <div style={{ maxWidth: "200px" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof TooltipTruncatedText>;

export const UsedInsideDataSet: Story = {};
