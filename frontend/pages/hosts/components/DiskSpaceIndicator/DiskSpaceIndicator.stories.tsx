import { Meta, StoryObj } from "@storybook/react";

import DiskSpaceIndicator from "./DiskSpaceIndicator";

const meta: Meta<typeof DiskSpaceIndicator> = {
  title: "Components/DiskSpaceIndicator",
  component: DiskSpaceIndicator,
  args: {
    gigsDiskSpaceAvailable: 100,
    percentDiskSpaceAvailable: 75,
    platform: "darwin",
  },
};

export default meta;

type Story = StoryObj<typeof DiskSpaceIndicator>;

export const Basic: Story = {};
