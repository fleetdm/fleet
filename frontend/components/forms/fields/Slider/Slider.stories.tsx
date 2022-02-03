import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import Slider from ".";

import "../../../../index.scss";

interface ISliderProps {
  value: boolean;
  inactiveText: string;
  activeText: string;
  onChange: () => void;
}

export default {
  component: Slider,
  title: "Components/FormFields/Slider",
  args: {
    value: false,
    inactiveText: "Off",
    activeText: "On",
    onChange: noop,
  },
} as Meta;

const Template: Story<ISliderProps> = (props) => <Slider {...props} />;

export const Default = Template.bind({});
