import { Meta, StoryObj } from "@storybook/react";
import React from "react";

import ViewAllHostsButton from ".";

const meta: Meta<typeof ViewAllHostsButton> = {
  component: ViewAllHostsButton,
  title: "Components/ViewAllHostsButton",
  argTypes: {
    condensed: { control: "boolean" },
    excludeChevron: { control: "boolean" },
    responsive: { control: "boolean" },
    rowHover: { control: "boolean" },
    customText: { control: "text" },
  },
  args: {
    // Stops Storybook from calling browserHistory.push and navigating away.
    noLink: true,
  },
};

export default meta;

type Story = StoryObj<typeof ViewAllHostsButton>;

export const Default: Story = {};

export const Condensed: Story = {
  args: { condensed: true },
};

export const NoChevron: Story = {
  args: { excludeChevron: true },
};

export const CustomText: Story = {
  args: { customText: "See all matching hosts" },
};

export const RowHover: Story = {
  name: "Row-hover reveal (hover the container)",
  args: { rowHover: true },
  render: (args) => (
    <div
      style={{
        padding: 12,
        border: "1px dashed var(--ui-fleet-black-25)",
        borderRadius: 4,
      }}
    >
      <ViewAllHostsButton {...args} />
    </div>
  ),
};
