import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import Button from ".";
import { IButtonProps } from "./Button";

import "../../../index.scss";

export default {
  component: Button,
  title: "Components/Button",
  argTypes: {
    variant: {
      options: [
        "brand",
        "success",
        "alert",
        "blue-green",
        "grey",
        "warning",
        "link",
        "label",
        "text-link",
        "text-icon",
        "inverse",
        "inverse-alert",
        "block",
        "unstyled",
        "unstyled-modal-query",
        "contextual-nav-item",
        "small-text-icon",
      ],
      control: "select",
    },
    type: {
      options: ["button", "submit", "reset"],
      control: "select",
    },
  },
  args: {
    autofocus: false,
    className: "",
    size: "",
    tabIndex: 0,
    title: "",
    onClick: noop,
  },
} as Meta;

const Template: Story<IButtonProps> = (props) => (
  <Button {...props}>Click Here</Button>
);

export const Default = Template.bind({});
Default.args = { variant: "brand", type: "button" };

export const Disabled = Template.bind({});
Disabled.args = { ...Default.args, disabled: true };
