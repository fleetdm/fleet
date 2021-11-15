import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import Modal from ".";
import { IModalProps } from "./Modal";

import "../../index.scss";

export default {
  component: Modal,
  title: "Components/Modal",
  args: {
    title: "Test modal",
    className: "",
    onExit: noop,
  },
} as Meta;

const Template: Story<IModalProps> = (props) => (
  <Modal {...props}>
    <div>This is a test description with lots of information.</div>
  </Modal>
);

export const Default = Template.bind({});
