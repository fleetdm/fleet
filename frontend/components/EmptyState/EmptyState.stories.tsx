import React from "react";
import type { Meta, StoryObj } from "@storybook/react";

import Button from "components/buttons/Button";
import EmptyState from "./EmptyState";

const meta: Meta<typeof EmptyState> = {
  title: "Components/EmptyState",
  component: EmptyState,
  argTypes: {
    width: {
      control: "select",
      options: ["default", "small"],
    },
  },
};

export default meta;

type Story = StoryObj<typeof EmptyState>;

export const Default: Story = {
  args: {
    header: "No reports",
    info: "Create a report to get started.",
    primaryButton: <Button>Create report</Button>,
  },
};

export const WithGraphic: Story = {
  args: {
    header: "No hosts",
    info: "Add your first host to get started with Fleet.",
    graphicName: "empty-hosts",
    primaryButton: <Button>Add hosts</Button>,
  },
};

export const WithAdditionalInfo: Story = {
  args: {
    header: "Additional configuration required",
    info:
      "Turn on MDM and automatic enrollment to deploy a custom bootstrap package.",
    additionalInfo: "Supported on macOS.",
    primaryButton: <Button>Turn on</Button>,
  },
};

export const WithTwoButtons: Story = {
  args: {
    header: "No policies",
    info: "Start monitoring compliance by creating your first policy.",
    primaryButton: <Button>Create policy</Button>,
    secondaryButton: (
      <Button variant="inverse">Import from library</Button>
    ),
  },
};

export const Small: Story = {
  args: {
    header: "No software",
    info: "Add software to install on hosts in this fleet.",
    primaryButton: <Button>Add software</Button>,
    width: "small",
  },
};

export const MinimalContent: Story = {
  args: {
    header: "Nothing to show",
  },
};

export const InfoOnly: Story = {
  args: {
    info: "Policies are not supported for this host.",
  },
};
