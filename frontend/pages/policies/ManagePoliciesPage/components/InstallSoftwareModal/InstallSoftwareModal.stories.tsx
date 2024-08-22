import { Meta, StoryObj } from "@storybook/react";

import InstallSoftwareModal from "./InstallSoftwareModal";

const meta: Meta<typeof InstallSoftwareModal> = {
  title: "Components/InstallSoftwareModal",
  component: InstallSoftwareModal,
};

export default meta;

type Story = StoryObj<typeof InstallSoftwareModal>;

export const Basic: Story = {};
