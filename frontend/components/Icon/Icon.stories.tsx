import { Meta, StoryObj } from "@storybook/react";

import { ICON_MAP } from "components/icons";

import Icon from ".";

const meta: Meta<typeof Icon> = {
  title: "Components/Icon",
  component: Icon,
  args: { name: "plus" },
  argTypes: {
    name: {
      control: { type: "select" },
      options: Object.keys(ICON_MAP).sort(),
    },
  },
};

export default meta;

type Story = StoryObj<typeof Icon>;

export const Basic: Story = {};
