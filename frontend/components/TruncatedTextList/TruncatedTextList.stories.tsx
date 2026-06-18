import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import TruncatedTextList from "./TruncatedTextList";

const meta: Meta<typeof TruncatedTextList> = {
  title: "Components/TruncatedTextList",
  component: TruncatedTextList,
  decorators: [
    (Story) => (
      <div style={{ width: 360, border: "1px dashed #ccc", padding: 8 }}>
        <Story />
      </div>
    ),
  ],
  args: {
    items: [
      "Engineering",
      "Product",
      "Quality Assurance",
      "Marketing",
      "Sales",
      "Support",
      "Operations",
    ],
  },
};

export default meta;

type Story = StoryObj<typeof TruncatedTextList>;

export const Basic: Story = {};

export const NarrowContainer: Story = {
  decorators: [
    (Story) => (
      <div style={{ width: 180, border: "1px dashed #ccc", padding: 8 }}>
        <Story />
      </div>
    ),
  ],
};

export const AllFit: Story = {
  args: { items: ["Mac", "Linux"] },
};

export const Empty: Story = {
  args: { items: [] },
};
