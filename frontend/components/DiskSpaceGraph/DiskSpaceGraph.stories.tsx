import { Meta, StoryObj } from "@storybook/react";

import DiskSpaceGraph from "./DiskSpaceGraph";

const meta: Meta<typeof DiskSpaceGraph> = {
  title: "Components/DiskSpaceGraph",
  component: DiskSpaceGraph,
  args: {
    baseClass: "disk-space-graph",
    gigsDiskSpaceAvailable: 100,
    percentDiskSpaceAvailable: 75,
    id: "disk-space-graph",
    platform: "darwin",
  },
};

export default meta;

type Story = StoryObj<typeof DiskSpaceGraph>;

export const Basic: Story = {};
