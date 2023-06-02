import { Meta, StoryObj } from "@storybook/react";

import SectionHeader from ".";

const meta: Meta<typeof SectionHeader> = {
  title: "Components/SectionHeader",
  component: SectionHeader,
  args: { title: "Section header title" },
};

export default meta;

type Story = StoryObj<typeof SectionHeader>;

export const Basic: Story = {};
