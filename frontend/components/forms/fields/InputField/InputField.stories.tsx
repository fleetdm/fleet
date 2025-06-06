import { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

// @ts-ignore
import InputField from ".";

import "../../../../index.scss";

const meta: Meta<typeof InputField> = {
  component: InputField,
  title: "Components/FormFields/Input",
  argTypes: {
    type: {
      control: "select",
      options: ["text", "password", "email", "number", "textarea"],
    },
    autofocus: { control: "boolean" },
    readOnly: { control: "boolean" },
    disabled: { control: "boolean" },
    blockAutoComplete: { control: "boolean" },
    enableCopy: { control: "boolean" },
    enableShowSecret: { control: "boolean" },
    copyButtonPosition: {
      control: "radio",
      options: ["inside", "outside"],
    },
    labelTooltipPosition: {
      control: "select",
      options: ["top", "right", "bottom", "left"],
    },
  },
};

export default meta;

type Story = StoryObj<typeof InputField>;

export const Default: Story = {
  args: {
    name: "default-input",
    label: "Default Input",
    placeholder: "Type here...",
    value: "",
    onChange: action("onChange"),
    onFocus: action("onFocus"),
    onBlur: action("onBlur"),
  },
};

export const WithError: Story = {
  args: {
    ...Default.args,
    name: "error-input",
    label: "Input with Error",
    error: "This field is required",
    value: "",
  },
};

export const WithHelpText: Story = {
  args: {
    ...Default.args,
    name: "help-text-input",
    label: "Input with Help Text",
    helpText: "This is some helpful information about the input field.",
  },
};

export const WithTooltip: Story = {
  args: {
    ...Default.args,
    name: "tooltip-input",
    label: "Input with Tooltip",
    tooltip: "This is additional information in a tooltip.",
    labelTooltipPosition: "right",
  },
};

export const Password: Story = {
  args: {
    ...Default.args,
    name: "password-input",
    label: "Password Input",
    type: "password",
    placeholder: "Enter password",
  },
};

export const ReadOnly: Story = {
  args: {
    ...Default.args,
    name: "readonly-input",
    label: "Read-only Input",
    readOnly: true,
    value: "This is read-only content",
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    name: "disabled-input",
    label: "Disabled Input",
    disabled: true,
    value: "This input is disabled",
  },
};

export const WithCopyButton: Story = {
  args: {
    ...Default.args,
    name: "copy-input",
    label: "Input with Copy Button",
    value: "Click to copy this text",
    enableCopy: true,
  },
};

export const Textarea: Story = {
  args: {
    ...Default.args,
    name: "textarea-input",
    label: "Textarea Input",
    type: "textarea",
    placeholder: "Enter multiple lines of text...",
  },
};
