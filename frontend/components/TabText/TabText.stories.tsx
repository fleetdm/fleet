import { Meta, StoryObj } from "@storybook/react";

import TabText from "./TabText";

const meta: Meta<typeof TabText> = {
  component: TabText,
  title: "Components/TabText",
  argTypes: {
    countVariant: {
      control: { type: "select" },
      options: [undefined, "alert", "pending"],
    },
    count: { control: "number" },
    showCheck: { control: "boolean" },
    children: { control: "text" },
  },
  args: {
    children: "Tab label",
  },
};

export default meta;

type Story = StoryObj<typeof TabText>;

export const Default: Story = {};

export const WithCount: Story = {
  args: { count: 42 },
};

export const AlertCount: Story = {
  args: { count: 7, countVariant: "alert" },
};

export const PendingCount: Story = {
  args: { count: 3, countVariant: "pending" },
};

export const WithCheck: Story = {
  args: { showCheck: true, children: "Configured" },
};

export const LargeCount: Story = {
  args: { count: 12345 },
};
