import { Meta, StoryObj } from "@storybook/react";

import { PLATFORM_NAME_TO_ICON_MAP } from "..";

import OSIcon from ".";

const meta: Meta<typeof OSIcon> = {
  title: "Components/Icon/OSIcon",
  component: OSIcon,
  argTypes: {
    name: {
      control: "select",
      options: Object.keys(PLATFORM_NAME_TO_ICON_MAP),
    },
    size: {
      control: "select",
      options: ["xsmall", "small", "medium", "large", "xlarge"],
    },
    url: { control: "text" },
  },
};

export default meta;

type Story = StoryObj<typeof OSIcon>;

export const Default: Story = {
  args: {
    name: "ubuntu",
    size: "medium",
  },
};
