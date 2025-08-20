import { Meta, StoryObj } from "@storybook/react";

import ScriptBatchDetails from "./ScriptBatchDetails";

const meta: Meta<typeof ScriptBatchDetails> = {
  title: "Components/ScriptBatchDetails",
  component: ScriptBatchDetails,
};

export default meta;

type Story = StoryObj<typeof ScriptBatchDetails>;

export const Basic: Story = {};
