import { Meta, StoryObj } from "@storybook/react";

import SetupScriptStatusCell from "./SetupScriptStatusCell";

const meta: Meta<typeof SetupScriptStatusCell> = {
  title: "Components/TableContainer/SetupScriptStatusCell",
  component: SetupScriptStatusCell,
};

export default meta;

type Story = StoryObj<typeof SetupScriptStatusCell>;

export const Pending: Story = {
  args: { status: "pending" },
};

export const Running: Story = {
  args: { status: "running" },
};

export const Success: Story = {
  args: { status: "success" },
};

export const Failure: Story = {
  args: { status: "failure" },
};

export const Cancelled: Story = {
  args: { status: "cancelled" },
};
