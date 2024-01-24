import { Meta, StoryObj } from "@storybook/react";

import DeviceUserError from "./DeviceUserError";

const meta: Meta<typeof DeviceUserError> = {
  title: "Components/DeviceUserError",
  component: DeviceUserError,
};

export default meta;

type Story = StoryObj<typeof DeviceUserError>;

export const Basic: Story = {};
