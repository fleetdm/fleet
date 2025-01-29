import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import Slider from ".";

import "../../../../index.scss";

const meta: Meta<typeof Slider> = {
  component: Slider,
  title: "Components/FormFields/Slider",
  args: {
    value: false,
    inactiveText: "Off",
    activeText: "On",
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof Slider>;

export const Default: Story = {};
