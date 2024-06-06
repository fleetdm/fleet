import { Meta, StoryObj } from "@storybook/react";

import StatusIndicator from "./StatusIndicator";

const meta: Meta<typeof StatusIndicator> = {
  title: "Components/StatusIndicator",
  component: StatusIndicator,
  args: {
    value: "100",
    tooltip: {
      tooltipText: "Tooltip text",
    },
  },
};

export default meta;

type Story = StoryObj<typeof StatusIndicator>;

export const Basic: Story = {};
