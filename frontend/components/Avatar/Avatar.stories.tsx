import { Meta, StoryObj } from "@storybook/react";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

import Avatar from ".";

const meta: Meta<typeof Avatar> = {
  component: Avatar,
  title: "Components/Avatar",
  args: {
    user: { gravatar_url: DEFAULT_GRAVATAR_LINK },
  },
  parameters: {
    design: {
      type: "figma",
      url:
        "https://www.figma.com/file/qbjRu8jf01BzEfdcge1dgu/Fleet-style-guide-2022-(WIP)?node-id=210-11078",
    },
  },
};

export default meta;

type Story = StoryObj<typeof Avatar>;

export const Default: Story = {};

export const Small: Story = {
  args: {
    size: "small",
  },
};

export const UseFleetAvatar: Story = {
  args: {
    useFleetAvatar: true,
  },
};
