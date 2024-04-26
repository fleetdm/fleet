import { Meta, StoryObj } from "@storybook/react";

import InheritedBadge from "./InheritedBadge";

const meta: Meta<typeof InheritedBadge> = {
  title: "Components/InheritedBadge",
  component: InheritedBadge,
};

export default meta;

type Story = StoryObj<typeof InheritedBadge>;

export const Basic: Story = {};
