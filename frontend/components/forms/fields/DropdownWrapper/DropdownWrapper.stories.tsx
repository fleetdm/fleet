// stories/DropdownWrapper.stories.tsx

import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import DropdownWrapper, { CustomOptionType } from "./DropdownWrapper";

// Define metadata for the story
const meta: Meta<typeof DropdownWrapper> = {
  title: "Components/DropdownWrapper",
  component: DropdownWrapper,
  argTypes: {
    onChange: { action: "changed" },
  },
  // Padding added to view tooltips
  decorators: [
    (Story) => (
      <div style={{ overflow: "visible", padding: "150px" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof DropdownWrapper>;

// Sample options to be used in the dropdown
const sampleOptions: CustomOptionType[] = [
  {
    label: "Option 1 - just help text",
    value: "option1",
    helpText: "Help text for option 1",
  },
  {
    label: "Option 2 - just tooltip",
    value: "option2",
    tooltipContent: "Tooltip for option 2",
  },
  { label: "Option 3 - just disabled", value: "option3", isDisabled: true },
  {
    label: "Option 4 - help text, disabled, and tooltip",
    value: "option4",
    helpText: "Help text for option 4",
    isDisabled: true,
    tooltipContent: "Tooltip for option 4",
  },
];

// Default story
export const Default: Story = {
  args: {
    options: sampleOptions,
    name: "dropdown-example",
    label: "Select an option",
  },
};

// Disabled story
export const Disabled: Story = {
  args: {
    ...Default.args,
    isDisabled: true,
  },
};

// With Help Text story
export const WithHelpText: Story = {
  args: {
    ...Default.args,
    helpText: "This is some help text for the dropdown",
  },
};

// With Error story
export const WithError: Story = {
  args: {
    ...Default.args,
    error: "This is an error message",
  },
};
