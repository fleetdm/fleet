import { Meta, StoryObj } from "@storybook/react";

import SaveAsNewQueryModal from "./SaveAsNewQueryModal";

const meta: Meta<typeof SaveAsNewQueryModal> = {
  title: "Components/SaveAsNewQueryModal",
  component: SaveAsNewQueryModal,
};

export default meta;

type Story = StoryObj<typeof SaveAsNewQueryModal>;

export const Basic: Story = {};
