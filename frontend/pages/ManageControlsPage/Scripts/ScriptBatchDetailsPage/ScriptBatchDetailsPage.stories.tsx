import { Meta, StoryObj } from "@storybook/react";

import ScriptBatchDetailsPage from "./ScriptBatchDetailsPage";

const meta: Meta<typeof ScriptBatchDetailsPage> = {
  title: "Components/ScriptBatchDetailsPage",
  component: ScriptBatchDetailsPage,
};

export default meta;

type Story = StoryObj<typeof ScriptBatchDetailsPage>;

export const Basic: Story = {};
