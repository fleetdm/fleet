import { Meta, StoryObj } from "@storybook/react";

import QueryFrequencyIndicator from "./QueryFrequencyIndicator";

const meta: Meta<typeof QueryFrequencyIndicator> = {
  title: "Components/QueryFrequencyIndicator",
  component: QueryFrequencyIndicator,
  args: {
    frequency: 300,
    checked: true,
  },
};

export default meta;

type Story = StoryObj<typeof QueryFrequencyIndicator>;

export const Basic: Story = {};
