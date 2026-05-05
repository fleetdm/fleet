import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import TooltipWrapper from ".";

import "../../index.scss";

const meta: Meta<typeof TooltipWrapper> = {
  component: TooltipWrapper,
  title: "Components/TooltipWrapper",
  argTypes: {
    position: {
      options: [
        "top",
        "top-start",
        "top-end",
        "right",
        "right-start",
        "right-end",
        "bottom",
        "bottom-start",
        "bottom-end",
        "left",
        "left-start",
        "left-end",
      ],
      control: "radio",
    },
  },
};

export default meta;

type Story = StoryObj<typeof TooltipWrapper>;

export const Default: Story = {
  args: {
    tipContent: "This is an example tooltip.",
    children: "Example text",
  },
  decorators: [
    (Story) => (
      <div style={{ margin: "4rem 0" }}>
        <Story />
      </div>
    ),
  ],
};
