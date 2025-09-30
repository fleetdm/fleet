import { Meta, StoryObj } from "@storybook/react";

import ConfirmRunScriptModal from "./ConfirmRunScriptModal";

const meta: Meta<typeof ConfirmRunScriptModal> = {
  title: "Components/ConfirmRunScriptModal",
  component: ConfirmRunScriptModal,
};

export default meta;

type Story = StoryObj<typeof ConfirmRunScriptModal>;

export const Basic: Story = {};
