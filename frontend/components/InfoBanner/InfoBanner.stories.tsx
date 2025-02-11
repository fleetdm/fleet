import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import InfoBanner from ".";

import "../../index.scss";

const meta: Meta<typeof InfoBanner> = {
  component: InfoBanner,
  title: "Components/InfoBanner",
};

export default meta;

type Story = StoryObj<typeof InfoBanner>;

export const Default: Story = {
  args: {
    children: <div>This is an Info Banner.</div>,
  },
};
