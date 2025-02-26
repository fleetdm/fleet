import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import Modal from ".";

import "../../index.scss";

const meta: Meta<typeof Modal> = {
  component: Modal,
  title: "Components/Modal",
  args: {
    title: "Test modal",
    className: "",
    onExit: noop,
  },
};

export default meta;

type Story = StoryObj<typeof Modal>;

export const Default: Story = {
  decorators: [
    (Story) => (
      <div style={{ height: "300px" }}>
        <Story />
      </div>
    ),
  ],
  args: {
    children: <div>This is a test description with lots of information.</div>,
  },
};
