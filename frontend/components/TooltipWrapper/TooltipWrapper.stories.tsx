import React from "react";
import { Meta, Story } from "@storybook/react";

import TooltipWrapper from ".";

import "../../index.scss";

interface ITooltipWrapperProps {
  children: React.ReactNode;
  tipContent: React.ReactNode;
}

export default {
  component: TooltipWrapper,
  title: "Components/TooltipWrapper",
  args: {
    tipContent: "This is an example tooltip.",
  },
  argTypes: {
    position: {
      options: [
        "top",
        "top-start",
        "top-end",
        "right",
        "right-start",
        "right-end",
        "bottom",
        "bottom-start",
        "bottom-end",
        "left",
        "left-start",
        "left-end",
      ],
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
    <br />
    <br />
    <br />
    <br />
  </>
);

export const Default = Template.bind({});
