import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";
import React, { useState } from "react";

import RevealButton from "./RevealButton";

const meta: Meta<typeof RevealButton> = {
  component: RevealButton,
  title: "Components/RevealButton",
  argTypes: {
    caretPosition: {
      control: { type: "select" },
      options: [undefined, "before", "after"],
    },
    disabled: { control: "boolean" },
    autofocus: { control: "boolean" },
    isShowing: { control: "boolean" },
  },
  args: {
    showText: "Show advanced options",
    hideText: "Hide advanced options",
    caretPosition: "after",
    onClick: noop,
  },
};

export default meta;

type Story = StoryObj<typeof RevealButton>;

export const Collapsed: Story = {
  args: { isShowing: false },
};

export const Expanded: Story = {
  args: { isShowing: true },
};

export const CaretBefore: Story = {
  args: { isShowing: false, caretPosition: "before" },
};

export const WithTooltip: Story = {
  args: {
    isShowing: false,
    tooltipContent: "Shows fields most users won't need to touch.",
  },
};

export const DisabledWithTooltip: Story = {
  args: {
    isShowing: false,
    disabled: true,
    disabledTooltipContent: "Available once you save the current form.",
  },
};

export const Interactive: Story = {
  render: (args) => {
    const [isShowing, setIsShowing] = useState(false);
    return (
      <RevealButton
        {...args}
        isShowing={isShowing}
        onClick={() => setIsShowing((v) => !v)}
      />
    );
  },
};
