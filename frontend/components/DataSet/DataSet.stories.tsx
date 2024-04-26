import { Meta, StoryObj } from "@storybook/react";

import DataSet from "./DataSet";

const meta: Meta<typeof DataSet> = {
  title: "Components/DataSet",
  component: DataSet,
  args: {
    title: "Data set",
    value: "This is the value",
  },
};

export default meta;

type Story = StoryObj<typeof DataSet>;

export const Basic: Story = {};
