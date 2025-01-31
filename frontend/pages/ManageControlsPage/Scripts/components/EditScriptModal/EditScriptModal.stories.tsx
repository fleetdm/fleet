import { Meta, StoryObj } from "@storybook/react";

import EditScriptModal from "./EditScriptModal";

const meta: Meta<typeof EditScriptModal> = {
  title: "Components/EditScriptModal",
  component: EditScriptModal,
};

export default meta;

type Story = StoryObj<typeof EditScriptModal>;

export const Basic: Story = {};
