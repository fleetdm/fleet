import { Meta, StoryObj } from "@storybook/react";

import TeamHostExpiryToggle from "./TeamHostExpiryToggle";

const meta: Meta<typeof TeamHostExpiryToggle> = {
  title: "Components/TeamHostExpiryToggle",
  component: TeamHostExpiryToggle,
};

export default meta;

type Story = StoryObj<typeof TeamHostExpiryToggle>;

export const Basic: Story = {};
