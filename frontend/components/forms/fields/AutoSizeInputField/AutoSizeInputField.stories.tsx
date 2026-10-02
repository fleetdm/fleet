import { Meta, StoryFn } from "@storybook/react";
import { noop } from "lodash";
import React from "react";

import { IAutoSizeInputFieldProps } from "./AutoSizeInputField";

import AutoSizeInputField from ".";

import "../../../../index.scss";

export default {
  component: AutoSizeInputField,
  title: "Components/FormFields/Input/AutoSizeInputField",
  args: {
    autofocus: false,
    disabled: false,
    isFocused: false,
    error: "",
    inputClassName: "",
    inputWrapperClass: "",
    inputOptions: "",
    name: "",
    placeholder: "Type here...",
    type: "",
    value: "",
    maxLength: 250,
    onFocus: noop,
    onChange: noop,
    onKeyPress: noop,
  },
} as Meta;

const Template: StoryFn<IAutoSizeInputFieldProps> = (props) => (
  <AutoSizeInputField {...props} />
);

export const Default = Template.bind({});
