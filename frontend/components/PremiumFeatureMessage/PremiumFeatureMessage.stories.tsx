import { Meta, StoryObj } from "@storybook/react";

import PremiumFeatureMessage from "./PremiumFeatureMessage";

const meta: Meta<typeof PremiumFeatureMessage> = {
  title: "Components/PremiumFeatureMessage",
  component: PremiumFeatureMessage,
  argTypes: {
    variant: {
      control: "radio",
      options: ["default", "compact"],
    },
  },
};

export default meta;

type Story = StoryObj<typeof PremiumFeatureMessage>;

/** Option B — bordered card container, used on pages and settings sections. */
export const Default: Story = {};

/** Option A — compact left-aligned layout, used inside modals. */
export const Compact: Story = {
  args: {
    variant: "compact",
  },
};
