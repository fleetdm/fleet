import { Meta, StoryObj } from "@storybook/react";

import Icon from ".";

const meta: Meta<typeof Icon> = {
  title: "Components/Icon",
  component: Icon,
  args: { name: "plus" },
};

export default meta;

type Story = StoryObj<typeof Icon>;

export const Basic: Story = {};
