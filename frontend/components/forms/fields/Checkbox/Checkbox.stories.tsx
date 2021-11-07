import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import Checkbox from ".";

import { ICheckboxProps } from "./Checkbox";

import "../../../../index.scss";

export default {
  component: Checkbox,
  title: "Components/FormFields/Checkbox",
  args: {
    value: false,
    disabled: false,
    indeterminate: false,
    className: "",
    name: "",
    wrapperClassName: "",
    onChange: noop,
  },
} as Meta;

const Template: Story<ICheckboxProps> = (props) => <Checkbox {...props} />;

export const Default = Template.bind({});
