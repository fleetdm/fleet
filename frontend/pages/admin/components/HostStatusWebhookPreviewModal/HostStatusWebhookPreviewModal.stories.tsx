import { Meta, StoryObj } from "@storybook/react";

import HostStatusWebhookPreviewModal from "./HostStatusWebhookPreviewModal";

const meta: Meta<typeof HostStatusWebhookPreviewModal> = {
  title: "Components/HostStatusWebhookPreviewModal",
  component: HostStatusWebhookPreviewModal,
  args: { isTeamScope: false },
};

export default meta;

type Story = StoryObj<typeof HostStatusWebhookPreviewModal>;

export const Basic: Story = {};
