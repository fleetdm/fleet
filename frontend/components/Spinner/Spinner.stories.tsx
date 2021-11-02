import React from "react";
import { Meta, Story } from "@storybook/react";

// @ts-ignore
import Spinner from ".";
import { ISpinnerProps } from "./Spinner";

import "../../index.scss";

export default {
  component: Spinner,
  title: "Components/Spinner",
  args: {
    isInButton: false,
  },
} as Meta;

const Template: Story<ISpinnerProps> = (props) => <Spinner {...props} />;

export const Default = Template.bind({});
