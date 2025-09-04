import { Meta, StoryObj } from "@storybook/react";

import SettingUpYourDevice from "./SettingUpYourDevice";

const meta: Meta<typeof SettingUpYourDevice> = {
  title: "Components/SettingUpYourDevice",
  component: SettingUpYourDevice,
};

export default meta;

type Story = StoryObj<typeof SettingUpYourDevice>;

export const Basic: Story = {};
