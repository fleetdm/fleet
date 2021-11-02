import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import Radio from ".";
import { IRadioProps } from "./Radio";

import "../../../../index.scss";

export default {
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
} as Meta;

const Template: Story<IRadioProps> = (props) => <Radio {...props} />;

export const Default = Template.bind({});
