import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import Avatar from "components/Avatar"; // @ts-ignore
import DropdownButton from ".";

import "../../../index.scss";

interface IOptions {
  disabled: boolean;
  label: string;
  onClick: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

interface IDropdownButtonProps {
  children: React.ReactChild;
  className?: string;
  disabled?: boolean;
  options: IOptions[];
  size?: string;
  tabIndex?: number;
  type?: string;
  variant?: string;
}

const options = [
  {
    label: "My account",
    onClick: noop,
  },
  {
    label: "Documentation",
    onClick: () =>
      window.open(
        "https://github.com/fleetdm/fleet/blob/main/docs/README.md",
        "_blank"
      ),
  },
  {
    label: "Sign out",
    onClick: noop,
  },
];

export default {
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
} as Meta;

const Template: Story<IDropdownButtonProps> = (props) => (
  <DropdownButton {...props}>
    <Avatar user={{ gravatarURL: DEFAULT_GRAVATAR_LINK }} size="small" />
  </DropdownButton>
);

export const Default = Template.bind({});

export const Disabled = Template.bind({});
Disabled.args = { ...Default.args, disabled: true };
