import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import IconToolTip from ".";
import { IIconToolTipProps } from "./IconToolTip";

import "../../index.scss";

export default {
  component: IconToolTip,
  title: 'Components/IconToolTip',
  args: {
    text: "This is a tooltip",
    isHtml: false,
    issue: false,
  }
} as Meta;

const Template: Story<IIconToolTipProps> = (props) => <IconToolTip {...props} />;

export const Default = Template.bind({});