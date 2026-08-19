import { Meta, StoryObj } from "@storybook/react";

import SetupSoftwareStatusCell from "./SetupSoftwareStatusCell";

const meta: Meta<typeof SetupSoftwareStatusCell> = {
  title: "Components/TableContainer/SetupSoftwareStatusCell",
  component: SetupSoftwareStatusCell,
};

export default meta;

type Story = StoryObj<typeof SetupSoftwareStatusCell>;

export const Pending: Story = {
  args: { status: "pending" },
};

export const Installing: Story = {
  args: { status: "running" },
};

export const Installed: Story = {
  args: { status: "success" },
};

export const Failure: Story = {
  args: { status: "failure" },
};
