import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Checkbox from ".";

const meta: Meta<typeof Checkbox> = {
  component: Checkbox,
  title: "Components/FormFields/Checkbox",
};

export default meta;

type Story = StoryObj<typeof Checkbox>;

export const Basic: Story = {};

export const WithLabel: Story = {
  args: {
    children: <b>Label</b>,
  },
};
