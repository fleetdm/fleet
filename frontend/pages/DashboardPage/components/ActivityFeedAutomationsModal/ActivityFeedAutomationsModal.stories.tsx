import { Meta, StoryObj } from "@storybook/react";

import ActivityFeedAutomationsModal from "./ActivityFeedAutomationsModal";

const meta: Meta<typeof ActivityFeedAutomationsModal> = {
  title: "Components/ActivityFeedAutomationsModal",
  component: ActivityFeedAutomationsModal,
};

export default meta;

type Story = StoryObj<typeof ActivityFeedAutomationsModal>;

export const Basic: Story = {};
