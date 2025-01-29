import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import SQLEditor from ".";

import "../../index.scss";

const meta: Meta<typeof SQLEditor> = {
  component: SQLEditor,
  title: "Components/SQLEditor",
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
    helpText: "",
    labelActionComponent: <></>,
    onLoad: noop,
    onChange: noop,
    handleSubmit: noop,
  },
};

export default meta;

type Story = StoryObj<typeof SQLEditor>;

export const Default: Story = {};
