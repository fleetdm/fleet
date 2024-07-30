import React, { KeyboardEvent } from "react";
import { Meta, StoryFn } from "@storybook/react";
import { noop } from "lodash";

import AutoSizeInputField from ".";

import "../../../../index.scss";

interface IAutoSizeInputFieldProps {
  name: string;
  placeholder: string;
  value: string;
  inputClassName?: string;
  maxLength: number;
  hasError?: boolean;
  isDisabled?: boolean;
  isFocused?: boolean;
  onFocus: () => void;
  onBlur: () => void;
  onChange: (newSelectedValue: string) => void;
  onKeyPress: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
}

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
