import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import InputField from ".";

import "../../../../index.scss";

const meta: Meta<typeof InputField> = {
  component: InputField,
  title: "Components/FormFields/Input",
  args: {
    autofocus: false,
    readOnly: false,
    disabled: false,
    error: "",
    inputClassName: "",
    inputWrapperClass: "",
    inputOptions: {},
    name: "",
    placeholder: "Type here...",
    type: "",
    value: "",
    onFocus: noop,
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof InputField>;

export const Default: Story = {};
