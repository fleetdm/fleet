import { Meta, StoryObj } from "@storybook/react";

import Graphic from "./";

const meta: Meta<typeof Graphic> = {
  title: "Components/Graphic",
  component: Graphic,
  args: { name: "empty-queries" },
};

export default meta;

type Story = StoryObj<typeof Graphic>;

export const Basic: Story = {};
