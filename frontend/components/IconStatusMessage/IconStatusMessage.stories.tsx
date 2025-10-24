import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import IconStatusMessage from ".";
import "../../index.scss";

const meta: Meta<typeof IconStatusMessage> = {
  component: IconStatusMessage,
  title: "Components/IconStatusMessage",
  argTypes: {
    message: {
      control: "text",
      description: "The message or content to display next to the icon",
    },
    iconName: {
      control: { type: "select" },
      options: [
        undefined,
        "success",
        "success-outline",
        "error",
        "error-outline",
        "info",
      ],
      description: "Name of the icon to display",
    },
    iconColor: {
      control: { type: "select" },
      options: [
        undefined,
        "core-fleet-green",
        "core-fleet-black",
        "ui-fleet-black-75",
        "status-success",
        "status-error",
      ],
      description: "Color of the icon element",
    },
    className: { control: "text" },
    testId: { control: "text" },
  },
  parameters: { controls: { expanded: true } },
};

export default meta;

type Story = StoryObj<typeof IconStatusMessage>;

export const Playground: Story = {
  args: {
    message: (
      <>
        Your <b>system update</b> was completed successfully.
      </>
    ),
    iconName: "success",
    iconColor: "core-fleet-green",
    className: "",
    testId: "icon-status-playground",
  },
  render: (args) => <IconStatusMessage {...args} />,
};
