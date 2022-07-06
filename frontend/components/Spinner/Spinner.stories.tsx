import React from "react";
import { Meta, Story } from "@storybook/react";

import Spinner from ".";

import "../../index.scss";

export default {
  component: Spinner,
  title: "Components/Spinner",
  args: {
    isInButton: false,
  },
} as Meta;

const Template: Story = (props) => <Spinner {...props} />;

export const Default = Template.bind({});
