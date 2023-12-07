import { Meta, StoryObj } from "@storybook/react";

import DiscardDataOption from "./DiscardDataOption";

const meta: Meta<typeof DiscardDataOption> = {
  title: "Components/DiscardDataOption",
  component: DiscardDataOption,
};

export default meta;

type Story = StoryObj<typeof DiscardDataOption>;

export const Basic: Story = {};
