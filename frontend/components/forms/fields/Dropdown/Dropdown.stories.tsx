import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import Dropdown from ".";

import "../../../../index.scss";

const authMethodOptions = [
  { label: "Plain", value: "authmethod_plain" },
  { label: "Cram MD5", value: "authmethod_cram_md5" },
  { label: "Login", value: "authmethod_login" },
];

const meta: Meta<typeof Dropdown> = {
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
};

export default meta;

type Story = StoryObj<typeof Dropdown>;

export const Default: Story = {};
