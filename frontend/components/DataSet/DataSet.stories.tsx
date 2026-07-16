import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import DataSet from "./DataSet";

const meta: Meta<typeof DataSet> = {
  title: "Components/DataSet",
  component: DataSet,
  args: {
    title: "Data set title",
    value: "This is the value",
  },
};

export default meta;

type Story = StoryObj<typeof DataSet>;

export const Basic: Story = {};

export const HorizontalOrientation: Story = {
  args: {
    orientation: "horizontal",
  },
};

// Multiline values wrap instead of truncating with an ellipsis. Use for
// free-form prose like "Resolve" or "Description".
export const Multiline: Story = {
  args: {
    title: "Resolve",
    value:
      "Re-enable FileVault on the device. Open System Settings, navigate to Privacy & Security, and turn on FileVault. Store the recovery key in a safe place.",
    multiline: true,
  },
  decorators: [
    (Story) => (
      <div style={{ maxWidth: "320px", padding: "16px" }}>{Story()}</div>
    ),
  ],
};
