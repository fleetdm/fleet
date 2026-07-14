import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import { AppContext, initialState } from "context/app";

import TeamsDropdown from ".";

const withAppContext = (isGlobalAdmin: boolean) => (
  Story: React.ComponentType
) => (
  <AppContext.Provider value={{ ...initialState, isGlobalAdmin }}>
    <div style={{ minHeight: 400 }}>
      <Story />
    </div>
  </AppContext.Provider>
);

const meta: Meta<typeof TeamsDropdown> = {
  title: "Components/TeamsDropdown",
  component: TeamsDropdown,
  args: {
    currentUserTeams: [
      { id: -1, name: "All fleets" },
      { id: 0, name: "No team" },
      { id: 1, name: "Servers" },
      { id: 2, name: "Servers (canary)" },
      { id: 3, name: "Workstations" },
      { id: 4, name: "Workstations (canary)" },
      { id: 5, name: "Company-owned mobile devices" },
    ],
    includeNoTeams: true,
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof TeamsDropdown>;

export const AsGlobalAdmin: Story = {
  decorators: [withAppContext(true)],
};

export const AsNonAdmin: Story = {
  decorators: [withAppContext(false)],
};
