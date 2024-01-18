import { Meta, StoryObj } from "@storybook/react";

import TeamSettings from "./TeamSettings";

const meta: Meta<typeof TeamSettings> = {
  title: "Components/TeamSettings",
  component: TeamSettings,
};

export default meta;

type Story = StoryObj<typeof TeamSettings>;

export const Basic: Story = {};
