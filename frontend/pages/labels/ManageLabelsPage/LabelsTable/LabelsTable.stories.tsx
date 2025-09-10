import { Meta, StoryObj } from "@storybook/react";

import LabelsTable from "./LabelsTable";

const meta: Meta<typeof LabelsTable> = {
  title: "Components/LabelsTable",
  component: LabelsTable,
};

export default meta;

type Story = StoryObj<typeof LabelsTable>;

export const Basic: Story = {};
