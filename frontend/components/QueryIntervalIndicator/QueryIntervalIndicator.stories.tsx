import { Meta, StoryObj } from "@storybook/react";

import QueryIntervalIndicator from "./QueryIntervalIndicator";

const meta: Meta<typeof QueryIntervalIndicator> = {
  title: "Components/QueryIntervalIndicator",
  component: QueryIntervalIndicator,
  args: {
    interval: 300,
    checked: true,
  },
};

export default meta;

type Story = StoryObj<typeof QueryIntervalIndicator>;

export const Basic: Story = {};
