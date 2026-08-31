import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Icon from "components/Icon";

import CopyButton from "./CopyButton";

const DEFAULT_ARGS = {
  copyText: "FLEET-MAINTAINED-APP-SLUG",
};

const meta: Meta<typeof CopyButton> = {
  component: CopyButton,
  title: "Components/CopyButton",
  argTypes: {
    variant: { control: false },
  },
  args: DEFAULT_ARGS,
};

export default meta;
type Story = StoryObj<typeof CopyButton>;

const inlineRow: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: 8,
  fontSize: 14,
};

export const SubduedVariant: Story = {
  args: { ...DEFAULT_ARGS, variant: "subdued" },
  render: (args) => (
    <span style={inlineRow}>
      <span>fleet-maintained-app-slug</span>
      <CopyButton {...args} />
    </span>
  ),
};

export const CompactVariant: Story = {
  args: { ...DEFAULT_ARGS, variant: "compact" },
  render: (args) => (
    <span style={inlineRow}>
      <span>fleet-maintained-app-slug</span>
      <CopyButton {...args} />
    </span>
  ),
};

export const SecondaryVariant: Story = {
  args: { ...DEFAULT_ARGS, variant: "secondary" },
  render: (args) => (
    <span style={inlineRow}>
      <code>SELECT * FROM users;</code>
      <CopyButton {...args}>
        Copy <Icon name="copy" />
      </CopyButton>
    </span>
  ),
};
