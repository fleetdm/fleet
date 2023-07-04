import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import TeamsDropdown from ".";

const meta: Meta<typeof TeamsDropdown> = {
  title: "Components/TeamsDropdown",
  component: TeamsDropdown,
  args: {
    currentUserTeams: [
      { id: 1, name: "Team 1" },
      { id: 2, name: "Team 2" },
    ],
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof TeamsDropdown>;

export const Basic: Story = {};
