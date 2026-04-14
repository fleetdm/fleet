import { Meta, StoryObj } from "@storybook/react";

import MDMCheckBuilder from "./MDMCheckBuilder";

const meta: Meta<typeof MDMCheckBuilder> = {
  title: "Components/MDMCheckBuilder",
  component: MDMCheckBuilder,
};

export default meta;

type Story = StoryObj<typeof MDMCheckBuilder>;

export const Basic: Story = {};
