import { Meta, StoryObj } from "@storybook/react";

import ConditionalAccess from "./ConditionalAccess";

const meta: Meta<typeof ConditionalAccess> = {
  title: "Components/ConditionalAccess",
  component: ConditionalAccess,
};

export default meta;

type Story = StoryObj<typeof ConditionalAccess>;

export const Basic: Story = {};
