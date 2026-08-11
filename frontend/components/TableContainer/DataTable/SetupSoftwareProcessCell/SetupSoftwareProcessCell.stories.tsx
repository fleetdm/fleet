import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";

import SetupSoftwareProcessCell from "./SetupSoftwareProcessCell";

// SoftwareIcon calls `useQuery` unconditionally (even when it doesn't fetch), so
// stories need a QueryClientProvider in scope or they throw "No QueryClient set".
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
type CustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: React.FC<CustomQueryClientProviderProps> = QueryClientProvider;

// Small inline SVG data URI to exercise the <img> render path (VPP apps or an
// app with an uploaded custom icon).
const iconURL =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24'%3E%3Crect width='24' height='24' rx='5' fill='%234a90d9'/%3E%3C/svg%3E";

const meta: Meta<typeof SetupSoftwareProcessCell> = {
  title: "Components/TableContainer/SetupSoftwareProcessCell",
  component: SetupSoftwareProcessCell,
  decorators: [
    (Story) => (
      <CustomQueryClientProvider client={queryClient}>
        <Story />
      </CustomQueryClientProvider>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof SetupSoftwareProcessCell>;

// Fleet-maintained app — renders its real matched brand icon (SVG fallback).
export const FleetMaintainedApp: Story = {
  args: { name: "Google Chrome" },
};

// Custom package with no matched icon — renders the generic package icon.
export const CustomPackage: Story = {
  args: { name: "Acme Corp Agent" },
};

// App with an uploaded icon URL (VPP / custom icon) — renders an <img>.
export const WithIconURL: Story = {
  args: { name: "Company Portal", url: iconURL },
};

// Real FMA brand icons, a custom package (generic icon), and a URL-based icon
// stacked together, to verify icons and "Install …" labels align across app
// types. Regression coverage for #46973.
export const MixedAlignment: Story = {
  render: () => (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <SetupSoftwareProcessCell name="Google Chrome" />
      <SetupSoftwareProcessCell name="1Password" />
      <SetupSoftwareProcessCell name="Visual Studio Code" />
      <SetupSoftwareProcessCell name="Zoom" />
      <SetupSoftwareProcessCell name="Acme Corp Agent" />
      <SetupSoftwareProcessCell name="Company Portal" url={iconURL} />
    </div>
  ),
};
