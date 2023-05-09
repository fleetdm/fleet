import { Meta, StoryObj } from "@storybook/react";

import TextCell from ".";

const meta: Meta<typeof TextCell> = {
  title: "Components/Table/TextCell",
  component: TextCell,
};

export default meta;

type Story = StoryObj<typeof TextCell>;

export const Basic: Story = {};
