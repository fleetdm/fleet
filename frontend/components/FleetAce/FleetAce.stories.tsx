import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import FleetAce from ".";

import "../../index.scss";

const meta: Meta<typeof FleetAce> = {
  component: FleetAce,
  title: "Components/FleetAce",
  args: {
    label: "Type some SQL here...",
    value: "SELECT 1 FROM TABLE_NAME;",
    readOnly: false,
    showGutter: false,
    wrapEnabled: false,
    fontSize: 16,
    name: "",
    error: "",
    wrapperClassName: "",
    helpText: "",
    labelActionComponent: <></>,
    onLoad: noop,
    onChange: noop,
    handleSubmit: noop,
  },
};

export default meta;

type Story = StoryObj<typeof FleetAce>;

export const Default: Story = {};
