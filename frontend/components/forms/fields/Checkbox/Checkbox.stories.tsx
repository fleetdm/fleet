import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

import Checkbox from ".";

const meta: Meta<typeof Checkbox> = {
  component: Checkbox,
  title: "Components/FormFields/Checkbox",
  argTypes: {
    value: {
      control: "boolean",
    },
    variant: {
      control: "select",
      options: ["default", "danger"],
    },
  },
};

export default meta;

type Story = StoryObj<typeof Checkbox>;

export const Basic: Story = {
  args: {
    onChange: action("onChange"),
  },
};

export const WithLabel: Story = {
  args: {
    children: <b>Label</b>,
  },
};
