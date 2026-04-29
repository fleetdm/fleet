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
    variant: {
      control: "select",
      options: [undefined, "list", "header-list", "form"],
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
    secondaryButton: <Button variant="inverse">Import from library</Button>,
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

export const List: Story = {
  args: {
    variant: "list",
    header: "No batch scripts started for this fleet",
    info: "When a script is run on multiple hosts, progress will appear here.",
  },
};

export const ListWithButton: Story = {
  args: {
    variant: "list",
    header: "You have no enroll secrets.",
    info: "Add secret(s) to enroll hosts.",
    primaryButton: <Button>Add secret</Button>,
  },
};

export const HeaderList: Story = {
  args: {
    variant: "header-list",
    header: "No scripts uploaded.",
  },
};

export const HeaderListWithButton: Story = {
  args: {
    variant: "header-list",
    header: "Add your certificate authority (CA)",
    info: "Help your end users connect to Wi-Fi or VPNs.",
    primaryButton: <Button>Add CA</Button>,
  },
};

export const Form: Story = {
  args: {
    variant: "form",
    header: "Additional configuration required",
    info: "To customize, first turn on automatic enrollment.",
    primaryButton: <Button>Turn on</Button>,
  },
};

export const FormWithLongInfo: Story = {
  args: {
    variant: "form",
    header: "Require end user authentication during setup",
    info: "Connect Fleet to your identity provider (IdP) to get started.",
    primaryButton: <Button>Connect</Button>,
  },
};
