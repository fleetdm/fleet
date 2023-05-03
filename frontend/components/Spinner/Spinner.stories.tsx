import { Meta, StoryObj } from "@storybook/react";

import Spinner from ".";

import "../../index.scss";

const meta: Meta<typeof Spinner> = {
  component: Spinner,
  title: "Components/Spinner",
};

export default meta;

type Story = StoryObj<typeof Spinner>;

export const SpinnerDefault: Story = {};

export const SpinnerSmall: Story = {
  args: {
    small: true,
  },
};

export const SpinnerButton: Story = {
  args: {
    button: true,
  },
};

export const SpinnerWhite: Story = {
  args: {
    white: true,
  },
};

export const SpinnerSize: Story = {
  argTypes: {
    size: {
      options: ["x-small", "small", "medium"],
      control: {
        type: "select",
        default: "medium",
      },
    },
  },
};
