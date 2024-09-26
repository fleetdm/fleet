import { Meta, StoryObj } from "@storybook/react";

import RunScriptModal from "./RunScriptModal";

const meta: Meta<typeof RunScriptModal> = {
  title: "Components/RunScriptModal",
  component: RunScriptModal,
};

export default meta;

type Story = StoryObj<typeof RunScriptModal>;

export const Basic: Story = {};
