import { Meta, StoryObj } from "@storybook/react";

import CancelScriptBatchModal from "./CancelScriptBatchModal";

const meta: Meta<typeof CancelScriptBatchModal> = {
  title: "Components/CancelScriptBatchModal",
  component: CancelScriptBatchModal,
};

export default meta;

type Story = StoryObj<typeof CancelScriptBatchModal>;

export const Basic: Story = {};
