import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import { AppContext, initialState } from "context/app";

import FleetsDropdown from ".";

// Realistic fleet names lifted from the design (Figma node 8105-799 in the
// Product design system file) so the stories match the visuals reviewers see.
const FLEETS_FEW = [
  { id: -1, name: "All fleets" },
  { id: 0, name: "Unassigned" },
  { id: 1, name: "Servers" },
  { id: 2, name: "Servers (canary)" },
  { id: 3, name: "Workstations" },
];

const FLEETS_MANY = [
  { id: -1, name: "All fleets" },
  { id: 0, name: "Unassigned" },
  { id: 1, name: "Servers" },
  { id: 2, name: "Servers (canary)" },
  { id: 3, name: "Workstations" },
  { id: 4, name: "Testing & QA" },
  { id: 5, name: "Employee-issued mobile devices" },
  { id: 6, name: "Personal mobile devices" },
  { id: 7, name: "IT servers" },
  { id: 8, name: "TV media centers" },
  { id: 9, name: "Smart fridges" },
];

const FLEETS_SCROLLABLE = [
  ...FLEETS_MANY,
  { id: 10, name: "Company-owned wearables" },
  { id: 11, name: "CEO exception devices" },
  { id: 12, name: "Company-owned mobile devices" },
  { id: 13, name: "Contractor-owned laptops" },
  { id: 14, name: "Regional office desktops" },
  { id: 15, name: "Kiosk terminals" },
  { id: 16, name: "Retail POS systems" },
];

const withAppContext = (isGlobalAdmin: boolean) => (
  Story: React.ComponentType
) => (
  <AppContext.Provider value={{ ...initialState, isGlobalAdmin }}>
    {/* Fixed 600px canvas so every story renders at a consistent height
        and the open menu has room to display below the trigger. */}
    <div style={{ height: 600 }}>
      <Story />
    </div>
  </AppContext.Provider>
);

const meta: Meta<typeof FleetsDropdown> = {
  title: "Components/FleetsDropdown",
  component: FleetsDropdown,
  args: {
    currentUserTeams: FLEETS_MANY,
    includeUnassigned: true,
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof FleetsDropdown>;

// ---------------------------------------------------------------------------
// Below the search threshold (<10 rows)
// ---------------------------------------------------------------------------

export const FewFleetsAsAdmin: Story = {
  name: "Few fleets — global admin (no search, footer only)",
  args: { currentUserTeams: FLEETS_FEW },
  decorators: [withAppContext(true)],
};

export const FewFleetsAsNonAdmin: Story = {
  name: "Few fleets — non-admin (no search, no footer)",
  args: { currentUserTeams: FLEETS_FEW },
  decorators: [withAppContext(false)],
};

// ---------------------------------------------------------------------------
// At the search threshold, still fits without scroll (10–14 rows)
// ---------------------------------------------------------------------------

export const ManyFleetsAsAdmin: Story = {
  name: "Many fleets — global admin (search + footer, no scroll)",
  args: { currentUserTeams: FLEETS_MANY },
  decorators: [withAppContext(true)],
};

export const ManyFleetsAsNonAdmin: Story = {
  name: "Many fleets — non-admin (search only, no scroll)",
  args: { currentUserTeams: FLEETS_MANY },
  decorators: [withAppContext(false)],
};

// ---------------------------------------------------------------------------
// Beyond the scroll threshold (15+ rows) — scroll-fade appears
// ---------------------------------------------------------------------------

export const ScrollableAsAdmin: Story = {
  name: "Scrollable list — global admin (search + fade + footer)",
  args: { currentUserTeams: FLEETS_SCROLLABLE },
  decorators: [withAppContext(true)],
};

export const ScrollableAsNonAdmin: Story = {
  name: "Scrollable list — non-admin (search + fade only)",
  args: { currentUserTeams: FLEETS_SCROLLABLE },
  decorators: [withAppContext(false)],
};

// ---------------------------------------------------------------------------
// Variants
// ---------------------------------------------------------------------------

export const AsFormField: Story = {
  name: "As form field (Save as new report modal)",
  args: {
    currentUserTeams: FLEETS_MANY,
    asFormField: true,
    includeAllFleets: false,
    selectedFleetId: 1,
  },
  decorators: [withAppContext(true)],
};

export const Disabled: Story = {
  args: {
    currentUserTeams: FLEETS_MANY,
    isDisabled: true,
  },
  decorators: [withAppContext(true)],
};

export const LongFleetName: Story = {
  name: "Long fleet name (trigger + option truncation)",
  args: {
    currentUserTeams: [
      { id: -1, name: "All fleets" },
      {
        id: 1,
        name:
          "Employee-issued mobile devices in the west-coast satellite offices",
      },
      { id: 2, name: "Workstations" },
      { id: 3, name: "Servers" },
    ],
    selectedFleetId: 1,
  },
  decorators: [withAppContext(true)],
};
