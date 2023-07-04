import { Meta, StoryObj } from "@storybook/react";

import PremiumFeatureMessage from "./PremiumFeatureMessage";

const meta: Meta<typeof PremiumFeatureMessage> = {
  title: "Components/PremiumFeatureMessage",
  component: PremiumFeatureMessage,
};

export default meta;

type Story = StoryObj<typeof PremiumFeatureMessage>;

export const Basic: Story = {};
