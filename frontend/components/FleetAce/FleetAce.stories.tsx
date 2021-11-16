import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import FleetAce from ".";
import { IFleetAceProps } from "./FleetAce";

import "../../index.scss";

export default {
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
    hint: "",
    labelActionComponent: <></>,
    onLoad: noop,
    onChange: noop,
    handleSubmit: noop,
  },
} as Meta;

const Template: Story<IFleetAceProps> = (props) => <FleetAce {...props} />;

export const Default = Template.bind({});
