import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Icon from "components/Icon";

import Tag from "./Tag";
import "../../index.scss";

const meta: Meta<typeof Tag> = {
  component: Tag,
  title: "Components/Tag",
  argTypes: {
    children: { control: "text" },
    size: { control: "radio", options: ["large", "small", "xsmall"] },
    disabled: { control: "boolean" },
    tooltip: { control: "text" },
    className: { control: "text" },
    onClick: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
  },
  parameters: { controls: { expanded: true } },
};

export default meta;

type Story = StoryObj<typeof Tag>;

export const Static: Story = {
  args: {
    children: "Inherited",
  },
};

export const Small: Story = {
  args: {
    children: "Patch",
    size: "small",
  },
};

export const XSmall: Story = {
  args: {
    children: "16 API endpoints",
    size: "xsmall",
  },
};

export const WithTooltip: Story = {
  args: {
    children: "Inherited",
    tooltip: "This report runs on all hosts.",
  },
};

export const WithIconAndText: Story = {
  args: {
    children: (
      <>
        <Icon size="small" name="warning" color="ui-fleet-black-75" />
        Report clipped
      </>
    ),
  },
};

export const Clickable: Story = {
  args: {
    type: "clickable",
    children: "iPadOS",
    onClick: () => undefined,
  },
};

export const ClickableDisabled: Story = {
  args: {
    type: "clickable",
    children: "iPadOS",
    disabled: true,
    onClick: () => undefined,
  },
};

export const Dismissible: Story = {
  args: {
    type: "dismissible",
    children: "Apple Silicon macOS hosts",
    onDismiss: () => undefined,
  },
};

export const DismissibleWithTooltip: Story = {
  args: {
    type: "dismissible",
    children: "Apple Silicon macOS hosts",
    tooltip: "Hosts filtered to Apple Silicon Macs.",
    dismissLabel: "Apple Silicon macOS hosts",
    onDismiss: () => undefined,
  },
};
