import { Meta, StoryObj } from "@storybook/react";

import InputFieldHiddenContent from ".";

const meta: Meta<typeof InputFieldHiddenContent> = {
  component: InputFieldHiddenContent,
  title: "Components/FormFields/InputFieldHiddenContent",
  args: {
    value: "test value",
  },
};

export default meta;

type Story = StoryObj<typeof InputFieldHiddenContent>;

export const Default: Story = {};
