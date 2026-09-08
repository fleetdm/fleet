import { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

import DeviceUserError from "./DeviceUserError";

const meta: Meta<typeof DeviceUserError> = {
  title: "Components/DeviceUserError",
  component: DeviceUserError,
};

export default meta;

type Story = StoryObj<typeof DeviceUserError>;

export const Default: Story = {};

export const InvalidDeviceURL: Story = {
  args: {
    isAuthenticationError: true,
  },
};

export const SSOSessionExpired: Story = {
  args: {
    ssoError: "session_expired",
  },
};

export const SSOCallbackFailed: Story = {
  args: {
    ssoError: "callback_failed",
  },
};

export const SSOSignInFailed: Story = {
  args: {
    ssoError: "sign_in_failed",
    onRetry: action("retried"),
  },
};
