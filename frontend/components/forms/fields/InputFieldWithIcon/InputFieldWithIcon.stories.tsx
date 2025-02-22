import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

// Import the InputFieldWithIcon component
import InputFieldWithIcon from ".";

import "../../../../index.scss";

const meta: Meta<typeof InputFieldWithIcon> = {
  component: InputFieldWithIcon,
  title: "Components/FormFields/InputFieldWithIcon",
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
    iconName: {
      options: [
        "chevrondown",
        "chevronleft",
        "chevronright",
        "chevronup",
        // Add more icon names here as needed
        "search",
        // Add other relevant icon names
      ],
      control: "select",
    },
  },
};

export default meta;

type Story = StoryObj<typeof InputFieldWithIcon>;

const Template: Story = (args) => <InputFieldWithIcon {...args} />;

export const Default: Story = Template.bind({});
Default.args = {
  label: "Email",
  placeholder: "Enter your email",
  type: "email",
  onChange: action("onChange"),
};

export const PasswordInput: Story = Template.bind({});
PasswordInput.args = {
  ...Default.args,
  iconName: "lock",
  label: "Password",
  placeholder: "Enter your password",
  type: "password",
};

export const WithError: Story = Template.bind({});
WithError.args = {
  ...Default.args,
  error: "Invalid email address",
};

export const WithHelpText: Story = Template.bind({});
WithHelpText.args = {
  ...Default.args,
  helpText: "We'll never share your email with anyone else.",
};

export const Disabled: Story = Template.bind({});
Disabled.args = {
  ...Default.args,
  disabled: true,
};

export const WithTooltip: Story = Template.bind({});
WithTooltip.args = {
  ...Default.args,
  tooltip: "Enter the email address associated with your account",
};

export const WithClearButton: Story = Template.bind({});
WithClearButton.args = {
  ...Default.args,
  clearButton: action("clearButton"),
  value: "example@email.com",
};

export const CustomIcon: Story = Template.bind({});
CustomIcon.args = {
  ...Default.args,
  iconSvg: "search", // Use an appropriate icon SVG name
  label: "Search",
  placeholder: "Search...",
};
