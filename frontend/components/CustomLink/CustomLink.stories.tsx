import { Meta, StoryObj } from "@storybook/react";

import CustomLink from ".";

const meta: Meta<typeof CustomLink> = {
  title: "Components/CustomLink",
  component: CustomLink,
};

export default meta;

type Story = StoryObj<typeof CustomLink>;

export const Basic: Story = {
  args: {
    url: "https://www.google.com",
    text: "Test Link",
  },
};

export const ExternalLink: Story = {
  args: {
    ...Basic.args,
    newTab: true,
  },
};
