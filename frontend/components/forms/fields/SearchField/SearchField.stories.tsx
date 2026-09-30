import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import SearchField from "./SearchField";

const meta: Meta<typeof SearchField> = {
  component: SearchField,
  title: "Components/SearchField",
  argTypes: {
    icon: {
      control: { type: "select" },
      options: ["search", "filter"],
    },
    clearButton: { control: "boolean" },
    disabled: { control: "boolean" },
  },
  args: {
    placeholder: "Search",
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof SearchField>;

export const Default: Story = {};

export const WithDefaultValue: Story = {
  args: { defaultValue: "web browser" },
};

export const WithClearButton: Story = {
  args: { defaultValue: "chrome", clearButton: true },
};

export const WithTooltip: Story = {
  args: {
    tooltip: "Searches by hostname, serial number, UUID, or hardware model.",
  },
};

export const FilterIcon: Story = {
  args: { placeholder: "Filter labels", icon: "filter" },
};

export const Disabled: Story = {
  args: { disabled: true, defaultValue: "read-only" },
};
