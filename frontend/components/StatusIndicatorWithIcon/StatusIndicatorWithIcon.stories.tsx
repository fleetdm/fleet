import { Meta, StoryObj } from "@storybook/react";

import StatusIndicatorWithIcon from "./StatusIndicatorWithIcon";

const meta: Meta<typeof StatusIndicatorWithIcon> = {
  title: "Components/StatusIndicatorWithIcon",
  component: StatusIndicatorWithIcon,
  args: {
    status: "success",
    value: "Yes",
    tooltip: {
      tooltipText: "Tooltip text",
    },
  },
};

export default meta;

type Story = StoryObj<typeof StatusIndicatorWithIcon>;

export const Basic: Story = {};
