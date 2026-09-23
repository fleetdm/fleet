import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";
import React from "react";

import { createMockActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import ActivityItem from "./ActivityItem";

const meta: Meta<typeof ActivityItem> = {
  component: ActivityItem,
  title: "Components/ActivityItem",
  argTypes: {
    isSoloActivity: { control: "boolean" },
    hideShowDetails: { control: "boolean" },
    hideCancel: { control: "boolean" },
    disableCancel: { control: "boolean" },
  },
  args: {
    activity: createMockActivity({
      actor_full_name: "Rachel Perkins",
      created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
    }),
    children: (
      <>
        <b>Rachel Perkins</b> edited agent options.
      </>
    ),
    onShowDetails: noop,
    onCancel: noop,
  },
};

export default meta;

type Story = StoryObj<typeof ActivityItem>;

export const Default: Story = {};

export const HideShowDetails: Story = {
  args: { hideShowDetails: true },
};

export const HideCancel: Story = {
  args: { hideCancel: true },
};

export const DisabledCancel: Story = {
  args: { disableCancel: false, hideCancel: false },
};

export const Solo: Story = {
  name: "Solo activity (details modal)",
  args: { isSoloActivity: true, hideCancel: true },
};

export const FleetInitiated: Story = {
  name: "Fleet-initiated (Fleet avatar)",
  args: {
    activity: createMockActivity({
      actor_full_name: "",
      actor_email: undefined,
      fleet_initiated: true,
      type: ActivityType.RanScript,
      created_at: new Date(Date.now() - 1000 * 60 * 60 * 3).toISOString(),
    }),
    children: (
      <>
        <b>Fleet</b> ran a script on <b>rachels-macbook</b>.
      </>
    ),
    hideCancel: true,
  },
};

export const ApiOnly: Story = {
  name: "API-only actor (bot avatar)",
  args: {
    activity: createMockActivity({
      actor_api_only: true,
      actor_full_name: "GitOps",
      actor_email: undefined,
    }),
    children: (
      <>
        <b>GitOps</b> edited a report.
      </>
    ),
    hideCancel: true,
  },
};
