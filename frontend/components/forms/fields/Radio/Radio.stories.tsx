import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import Radio from ".";

import "../../../../index.scss";

const meta: Meta<typeof Radio> = {
  component: Radio,
  title: "Components/FormFields/Radio",
  args: {
    checked: true,
    disabled: false,
    label: "Selected radio",
    value: "",
    id: "",
    name: "",
    className: "",
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof Radio>;

export const Default: Story = {};
