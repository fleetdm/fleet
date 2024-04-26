import { Meta, StoryObj } from "@storybook/react";

import SecretField from ".";

const meta: Meta<typeof SecretField> = {
  title: "Components/SecretField",
  component: SecretField,
  args: { secret: "secret" },
};

export default meta;

type Story = StoryObj<typeof SecretField>;

export const Basic: Story = {};
