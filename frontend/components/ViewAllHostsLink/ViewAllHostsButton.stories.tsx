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
  // The reveal rule lives at `.table-container tr:hover .row-hover-button`, so
  // the row + container are required for this story to actually demo the effect.
  name: "Row-hover reveal (hover the row)",
  args: { rowHover: true },
  render: (args) => (
    <div className="table-container">
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <tbody>
          <tr>
            <td style={{ padding: 12 }}>rachels-macbook.local</td>
            <td style={{ padding: 12, textAlign: "right" }}>
              <ViewAllHostsButton {...args} />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  ),
};
