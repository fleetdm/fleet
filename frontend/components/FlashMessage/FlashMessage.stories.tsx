import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import FlashMessage from ".";

import { IFlashMessage } from "./FlashMessage";

import "../../index.scss";

export default {
  component: FlashMessage,
  title: "Components/FlashMessage",
  argTypes: {
    fullWidth: {
      control: "boolean",
    },
    isPersistent: {
      control: "boolean",
    },
  },
  args: {
    fullWidth: true,
    isPersistent: true,
    notification: {
      message: "I am a message. Hear me roar!",
      alertType: "success",
      isVisible: true,
    },
  },
} as Meta;

const Template: Story<IFlashMessage> = (props) => <FlashMessage {...props} />;

export const Default = Template.bind({});
