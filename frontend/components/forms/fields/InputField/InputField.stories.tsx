import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import InputField from ".";

import "../../../../index.scss";

interface IInputFieldProps {
  autofocus?: boolean;
  disabled?: boolean;
  error?: string;
  inputClassName?: string;
  inputWrapperClass?: string;
  inputOptions?: Record<string, unknown>; // other html input props
  name?: string;
  placeholder: string;
  type?: string;
  value: string;
  onFocus?: () => void;
  onChange?: (value: string) => void;
}

export default {
  component: InputField,
  title: "Components/FormFields/Input",
  args: {
    autofocus: false,
    disabled: false,
    error: "",
    inputClassName: "",
    inputWrapperClass: "",
    inputOptions: "",
    name: "",
    placeholder: "Type here...",
    type: "",
    value: "",
    onFocus: noop,
    onChange: noop,
  },
} as Meta;

const Template: Story<IInputFieldProps> = (props) => <InputField {...props} />;

export const Default = Template.bind({});
