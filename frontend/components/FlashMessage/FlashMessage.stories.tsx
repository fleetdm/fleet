import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import FlashMessage from ".";

import "../../index.scss";

const meta: Meta<typeof FlashMessage> = {
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
};

export default meta;

type Story = StoryObj<typeof FlashMessage>;

export const Default: Story = {};
