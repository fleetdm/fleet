import { Meta, StoryObj } from "@storybook/react";

import ConditionalAccessModal from "./ConditionalAccessModal";

const meta: Meta<typeof ConditionalAccessModal> = {
  title: "Components/ConditionalAccessModal",
  component: ConditionalAccessModal,
};

export default meta;

type Story = StoryObj<typeof ConditionalAccessModal>;

export const Basic: Story = {};
