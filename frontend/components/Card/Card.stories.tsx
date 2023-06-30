import { Meta, StoryObj } from "@storybook/react";

import Card from ".";

const meta: Meta<typeof Card> = {
  component: Card,
  title: "Components/Card",
  args: {
    children: "card content",
  },
};

export default meta;

type Story = StoryObj<typeof Card>;

export const Default: Story = {};
