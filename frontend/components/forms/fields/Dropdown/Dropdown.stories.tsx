import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import { IDropdownOption } from "interfaces/dropdownOption"; // @ts-ignore
import Dropdown from ".";

import "../../../../index.scss";

interface IDropdownProps {
  className?: string;
  clearable?: boolean;
  searchable?: boolean;
  disabled?: boolean;
  error?: string;
  label?: string | string[];
  labelClassName?: string;
  multi?: boolean;
  name?: string;
  options: IDropdownOption[];
  placeholder?: string | string[];
  value?: string | string[] | number;
  wrapperClassName?: string;
  onChange: () => void;
  onOpen: () => void;
  onClose: () => void;
}

const authMethodOptions = [
  { label: "Plain", value: "authmethod_plain" },
  { label: "Cram MD5", value: "authmethod_cram_md5" },
  { label: "Login", value: "authmethod_login" },
];

export default {
  component: Dropdown,
  title: "Components/FormFields/Dropdown",
  args: {
    className: "",
    clearable: false,
    searchable: false,
    disabled: false,
    error: "",
    label: "",
    labelClassName: "",
    multi: false,
    name: "",
    options: authMethodOptions,
    placeholder: "Choose one...",
    value: "",
    wrapperClassName: "",
    onChange: noop,
    onOpen: noop,
    onClose: noop,
  },
} as Meta;

const Template: Story<IDropdownProps> = (props) => <Dropdown {...props} />;

export const Default = Template.bind({});
