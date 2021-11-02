import React from "react";
import { Meta, Story } from "@storybook/react";

// @ts-ignore
import Radio from ".";
import { IRadioProps } from "./Radio";

import "../../../../index.scss";

export default {
  component: Radio,
  title: 'Components/FormFields/Radio',
  args: {
    checked: true,
    disabled: false,
    label: "Selected radio",
    value: "",
    id: "",
    name: "",
    className: "",
    onChange: () => {},
  }
} as Meta;

const Template: Story<IRadioProps> = (props) => <Radio {...props} />;

export const Default = Template.bind({});