import { Meta, StoryObj } from "@storybook/react";

import CardHeader from ".";

const meta: Meta<typeof CardHeader> = {
  component: CardHeader,
  title: "Components/CardHeader",
  args: {
    header: "Card header",
    subheader: "This is a card subtitle",
  },
};

export default meta;

type Story = StoryObj<typeof CardHeader>;

export const Default: Story = {};
