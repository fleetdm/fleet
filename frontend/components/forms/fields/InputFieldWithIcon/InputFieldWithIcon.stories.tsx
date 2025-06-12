import type { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

// @ts-ignore
import InputFieldWithIcon from ".";

// Define metadata for the story
const meta: Meta<typeof InputFieldWithIcon> = {
  title: "Components/FormFields/InputFieldWithIcon",
  component: InputFieldWithIcon,
  argTypes: {
    type: {
      options: ["text", "password", "email", "number"],
      control: "select",
    },
    value: {
      control: "text",
    },
    disabled: {
      control: "boolean",
    },
    error: {
      control: "text",
    },
    helpText: {
      control: "text",
    },
    tooltip: {
      control: "text",
    },
    iconSvg: {
      options: ["search", "filter"], // Add more icons as needed
      control: "select",
    },
  },
};

export default meta;

type Story = StoryObj<typeof InputFieldWithIcon>;

// Default story
export const Default: Story = {
  args: {
    label: "Email",
    placeholder: "Enter your email",
    type: "email",
    onChange: action("onChange"),
  },
};

// Password input story
export const PasswordInput: Story = {
  args: {
    ...Default.args,
    label: "Password",
    placeholder: "Enter your password",
    type: "password",
  },
};

// With error story
export const WithError: Story = {
  args: {
    ...Default.args,
    error: "Invalid email address",
  },
};

// With help text story
export const WithHelpText: Story = {
  args: {
    ...Default.args,
    helpText: "We'll never share your email with anyone else.",
  },
};

// Disabled story
export const Disabled: Story = {
  args: {
    ...Default.args,
    disabled: true,
  },
};

// With tooltip story
export const WithTooltip: Story = {
  args: {
    ...Default.args,
    tooltip: "Enter the email address associated with your account",
  },
};

// With clear button story
export const WithClearButton: Story = {
  args: {
    ...Default.args,
    clearButton: action("clearButton"),
    value: "example@email.com",
  },
};

// Custom icon story
export const CustomIcon: Story = {
  args: {
    ...Default.args,
    iconSvg: "search", // Use an appropriate icon SVG name
    label: "Search",
    placeholder: "Search...",
  },
};
