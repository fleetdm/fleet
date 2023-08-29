import { Meta, StoryObj } from "@storybook/react";

import PlatformCell from ".";

const meta: Meta<typeof PlatformCell> = {
  title: "Components/Table/PlatformCell",
  component: PlatformCell,
  args: {
    platforms: ["darwin", "windows", "linux"],
  },
};

export default meta;

type Story = StoryObj<typeof PlatformCell>;

export const Basic: Story = {};
