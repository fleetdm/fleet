import React from "react";
import { Meta, Story } from "@storybook/react";

import TooltipWrapper from ".";

import "../../index.scss";

interface ITooltipWrapperProps {
  children: string;
  tipContent: string;
}

export default {
  component: TooltipWrapper,
  title: "Components/Tooltip",
  args: {
    tipContent: "This is ax example tooltip."
  },
} as Meta;

const Template: Story<ITooltipWrapperProps> = (props) => (
  <TooltipWrapper {...props}>Example text</TooltipWrapper>
);

export const Default = Template.bind({});
