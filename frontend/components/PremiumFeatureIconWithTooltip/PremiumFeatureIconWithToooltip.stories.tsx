import { Meta, StoryObj } from "@storybook/react";

import PremiumFeatureIconWithTooltip from "./PremiumFeatureIconWithTooltip";

const meta: Meta<typeof PremiumFeatureIconWithTooltip> = {
  title: "Components/PremiumFeatureIconWithTooltip",
  component: PremiumFeatureIconWithTooltip,
};

export default meta;

type Story = StoryObj<typeof PremiumFeatureIconWithTooltip>;

export const Basic: Story = {};
