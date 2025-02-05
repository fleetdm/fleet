import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import Button from ".";

import "../../../index.scss";

const meta: Meta<typeof Button> = {
  // TODO: change this after button is updated to a functional component. For
  // some reason the typing is incorrect becuase Button is a class component.
  component: Button as any,
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
        "oversized",
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
};

export default meta;

type Story = StoryObj<typeof Button>;

export const Default: Story = {
  args: {
    variant: "brand",
    type: "button",
    children: "Click Here",
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    disabled: true,
  },
};
