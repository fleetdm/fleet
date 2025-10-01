import { Meta, StoryObj } from "@storybook/react";

import BackButton from "./BackButton";

const meta: Meta<typeof BackButton> = {
  title: "Components/BackButton",
  component: BackButton,
  args: {
    text: "Back",
  },
};

export default meta;

type Story = StoryObj<typeof BackButton>;

export const Basic: Story = {};
