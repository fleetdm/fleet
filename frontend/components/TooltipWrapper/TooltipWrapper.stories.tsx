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
  title: "Components/NewTooltipWrapper",
  args: {
    tipContent: "This is an example tooltip.",
  },
  argTypes: {
    position: {
      options: ["top", "bottom"],
      control: "radio",
    },
  },
} as Meta;

// using line breaks to create space for top position
const Template: Story<ITooltipWrapperProps> = (props) => (
  <>
    <br />
    <br />
    <br />
    <br />
    <TooltipWrapper {...props}>Example text</TooltipWrapper>
  </>
);

export const Default = Template.bind({});
