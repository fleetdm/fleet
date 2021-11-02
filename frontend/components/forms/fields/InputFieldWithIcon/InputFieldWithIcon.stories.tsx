import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import InputFieldWithIcon from ".";

import "../../../../index.scss";

interface IInputFieldWithIconProps {
  autofocus?: boolean;
  error?: string;
  hint?: string | string[];
  iconName?: string;
  label?: string;
  name?: string;
  placeholder?: string;
  tabIndex?: number;
  type?: string;
  className?: string;
  disabled?: boolean;
  iconPosition?: "start" | "end";
  onChange?: () => void;
}

export default {
  component: InputFieldWithIcon,
  title: "Components/FormFields/InputWithIcon",
  argTypes: {
    iconPosition: {
      options: ["start", "end"],
      control: "radio",
    },
    iconName: {
      options: [
        "kolide-logo-flat",
        "chevrondown",
        "chevronleft",
        "chevronright",
        "chevronup",
        "cpu",
        "downcarat",
        "filter",
        "mac",
        "memory",
        "storage",
        "upcarat",
        "uptime",
        "world",
        "osquery",
        "join",
        "add-button",
        "packs",
        "help",
        "admin",
        "config",
        "mia",
        "success-check",
        "offline",
        "windows-original",
        "centos-original",
        "ubuntu-original",
        "apple-original",
        "search",
        "all-hosts",
        "alerts",
        "logout",
        "user-settings",
        "clipboard",
        "list-select",
        "grid-select",
        "label",
        "docker",
        "cloud",
        "self-hosted",
        "help-solid",
        "help-stroke",
        "warning-filled",
        "delete-cloud",
        "pdf",
        "credit-card-small",
        "billing-card",
        "lock-big",
        "link-big",
        "briefcase",
        "name-card",
        "kolide-logo",
        "business",
        "clock",
        "host-large",
        "single-host",
        "username",
        "password",
        "email",
        "hosts",
        "query",
        "import",
        "pencil",
        "add-plus",
        "x",
        "kill-kolide",
        "right-arrow",
        "camera",
        "plus-minus",
        "bold-plus",
        "linux-original",
        "clock2",
        "trash",
        "laptop-plus",
        "wrench-hand",
        "external-link",
        "fullscreen",
        "windowed",
        "heroku",
        "ubuntu",
        "windows",
        "centos",
        "apple",
        "linux",
      ],
      control: "select",
    },
  },
  args: {
    autofocus: false,
    iconPosition: "start",
    iconName: "email",
    disabled: false,
    label: "Email",
    placeholder: "Type here...",
    error: "",
    hint: "",
    name: "",
    tabIndex: "",
    type: "",
    className: "",
    onChange: noop,
  },
} as Meta;

const Template: Story<IInputFieldWithIconProps> = (props) => (
  <InputFieldWithIcon {...props} />
);

export const Default = Template.bind({});
