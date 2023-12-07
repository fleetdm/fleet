import { Meta, StoryObj } from "@storybook/react";

import LogDestinationIndicator from "./LogDestinationIndicator";

const meta: Meta<typeof LogDestinationIndicator> = {
  title: "Components/LogDestinationIndicator",
  component: LogDestinationIndicator,
  args: {
    logDestination: "filesystem",
  },
};

export default meta;

type Story = StoryObj<typeof LogDestinationIndicator>;

export const Basic: Story = {};
