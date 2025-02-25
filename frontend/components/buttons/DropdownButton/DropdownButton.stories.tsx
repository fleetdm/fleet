import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import Avatar from "components/Avatar";
// @ts-ignore
import DropdownButton from ".";

import "../../../index.scss";

const options = [
  {
    label: "My account",
    onClick: noop,
  },
  {
    label: "Documentation",
    onClick: () => window.open("https://fleetdm.com/docs", "_blank"),
  },
  {
    label: "Sign out",
    onClick: noop,
  },
];

const meta: Meta<typeof DropdownButton> = {
  component: DropdownButton,
  title: "Components/DropdownButton",
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
  parameters: {
    backgrounds: {
      default: "header",
      values: [
        {
          name: "header",
          value: "linear-gradient(270deg, #201e43 0%, #353d62 100%)",
        },
      ],
    },
  },
  args: {
    variant: "unstyled",
    className: "story",
    size: "",
    tabIndex: 0,
    options,
  },
};

export default meta;

type Story = StoryObj<typeof DropdownButton>;

export const Default: Story = {
  args: {
    children: (
      <Avatar user={{ gravatar_url: DEFAULT_GRAVATAR_LINK }} size="small" />
    ),
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    disabled: true,
  },
};
