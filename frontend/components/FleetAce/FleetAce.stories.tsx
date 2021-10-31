import React from "react";
import { Meta, Story } from "@storybook/react";

import FleetAce from ".";

import { IFleetAceProps } from "./FleetAce";

import "../../index.scss";

export default {
  component: FleetAce,
  title: 'Components/FleetAce',
  args: {
    error: "",
    fontSize: 16,
    label: "Type some SQL here...",
    name: "",
    value: "SELECT 1 FROM TABLE_NAME;",
    readOnly: false,
    showGutter: false,
    wrapEnabled: false,
    wrapperClassName: "",
    hint: "",
    labelActionComponent: <></>,
    onLoad: () => {},
    onChange: () => {},
    handleSubmit: () => {},
  }
} as Meta;

const Template: Story<IFleetAceProps> = (props) => <FleetAce {...props} />;

export const Default = Template.bind({});