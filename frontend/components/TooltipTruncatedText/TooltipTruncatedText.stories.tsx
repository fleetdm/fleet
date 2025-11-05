import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import TooltipTruncatedText from ".";
import DataSet from "../DataSet";
import Checkbox from "../forms/fields/Checkbox";

import "../../index.scss";

const meta: Meta<typeof TooltipTruncatedText> = {
  component: TooltipTruncatedText,
  title: "Components/TooltipTruncatedText",
  args: {
    value:
      "This is an example of a very long text that will be truncated in the display area.",
  },
  decorators: [
    (Story) => (
      <div style={{ maxWidth: "200px", padding: "150px", overflow: "visible" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof TooltipTruncatedText>;

export const Default: Story = {};

export const UsedInsideDataSet: Story = {
  decorators: [
    (Story) => (
      <DataSet
        className="my-dataset"
        title="Example Title"
        value={<Story />}
        orientation="horizontal"
      />
    ),
  ],
};
