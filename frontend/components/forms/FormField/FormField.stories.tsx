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
    label: "Field label",
  },
  // Autodocs stacks every story on one page; per-story `name` keeps each
  // label's `htmlFor` bound to its own input (see #53904 CodeRabbit review).
  // Building the child from args also lets the `disabled` control actually
  // toggle the input, not just the label styling.
  render: (args) => (
    <FormField {...args}>
      <input
        id={args.name}
        type="text"
        className="input-field"
        placeholder={args.disabled ? "Disabled" : "Type something"}
        disabled={args.disabled}
      />
    </FormField>
  ),
};

export default meta;

type Story = StoryObj<typeof FormField>;

export const Default: Story = {
  args: { name: "form-field-default" },
};

export const WithHelpText: Story = {
  args: {
    name: "form-field-help",
    helpText:
      "Shown under the input in small grey text. Use to clarify formatting or scope.",
  },
};

export const WithTooltip: Story = {
  args: {
    name: "form-field-tooltip",
    tooltip: "Extra context that appears on hover over the label.",
  },
};

export const WithError: Story = {
  args: { name: "form-field-error", error: "Enter a value" },
};

export const Disabled: Story = {
  args: {
    name: "form-field-disabled",
    disabled: true,
    helpText: "This field is disabled.",
  },
};
