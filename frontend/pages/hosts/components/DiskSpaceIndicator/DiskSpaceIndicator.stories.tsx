import { Meta, StoryObj } from "@storybook/react";

import DiskSpaceIndicator from "./DiskSpaceIndicator";

const meta: Meta<typeof DiskSpaceIndicator> = {
  title: "Components/DiskSpaceIndicator",
  component: DiskSpaceIndicator,
  args: {
    baseClass: "disk-space-indicator",
    gigsDiskSpaceAvailable: 100,
    percentDiskSpaceAvailable: 75,
    id: "disk-space-indicator",
    platform: "darwin",
  },
};

export default meta;

type Story = StoryObj<typeof DiskSpaceIndicator>;

export const Basic: Story = {};
