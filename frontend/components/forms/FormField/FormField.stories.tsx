import { Meta, StoryObj } from "@storybook/react";
import React from "react";

import FormField from "./FormField";

const meta: Meta<typeof FormField> = {
  component: FormField,
  title: "Components/FormField",
  argTypes: {
    labelTooltipPosition: {
      control: { type: "select" },
      options: [undefined, "top", "right", "bottom", "left"],
    },
    disabled: { control: "boolean" },
  },
  args: {
    name: "example",
    label: "Field label",
    children: (
      <input
        id="example"
        type="text"
        className="input-field"
        placeholder="Type something"
      />
    ),
  },
};

export default meta;

type Story = StoryObj<typeof FormField>;

export const Default: Story = {};

export const WithHelpText: Story = {
  args: {
    helpText:
      "Shown under the input in small grey text. Use to clarify formatting or scope.",
  },
};

export const WithTooltip: Story = {
  args: {
    tooltip: "Extra context that appears on hover over the label.",
  },
};

export const WithError: Story = {
  args: { error: "Enter a value" },
};

export const Disabled: Story = {
  args: {
    disabled: true,
    helpText: "This field is disabled.",
    children: (
      <input
        id="example"
        type="text"
        className="input-field"
        placeholder="Disabled"
        disabled
      />
    ),
  },
};
