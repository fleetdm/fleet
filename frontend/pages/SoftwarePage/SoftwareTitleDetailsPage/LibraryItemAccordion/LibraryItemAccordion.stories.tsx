import { Meta, StoryObj } from "@storybook/react";

import { ILabelSoftwareTitle } from "interfaces/label";

import LibraryItemAccordion from "./LibraryItemAccordion";

const labels7: ILabelSoftwareTitle[] = Array.from({ length: 7 }, (_, i) => ({
  id: i + 1,
  name: `Label ${i + 1}`,
})) as ILabelSoftwareTitle[];

const meta: Meta<typeof LibraryItemAccordion> = {
  title: "Pages/SoftwareTitleDetailsPage/LibraryItemAccordion",
  component: LibraryItemAccordion,
  args: {
    filename: "GoogleChrome.pkg",
    version: "149.0.7827.54",
    addedAt: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(),
    isActive: true,
    isLatest: true,
    labels: labels7,
    installed: 32,
    pending: 5,
    failed: 3,
    hashSha256:
      "af001543fcc5fbf484203b207d8af4fce44fc6975ca3db0eac49a49581af29b7",
    downloadUrl: "https://example.com/installer.pkg",
  },
};

export default meta;

type Story = StoryObj<typeof LibraryItemAccordion>;

export const Collapsed: Story = {};

export const Expanded: Story = {
  parameters: {
    docs: {
      description: {
        story:
          "Manually click the chevron in the Collapsed story to see the expanded panel. This entry is documentation-only since expansion is internal state.",
      },
    },
  },
};

export const LatestActive: Story = {
  args: {
    isLatest: true,
    isPinned: false,
  },
};

export const PinnedActive: Story = {
  args: {
    isLatest: false,
    isPinned: true,
  },
};

export const AllHostsNoLabels: Story = {
  args: {
    isLatest: true,
    labels: [],
  },
};

export const Inactive: Story = {
  args: {
    isActive: false,
    isLatest: false,
    isPinned: false,
    labels: [],
    version: "148.0.7778.179",
    addedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 20).toISOString(),
  },
};

export const TrashDisabledGitOps: Story = {
  args: {
    trashDisabled: true,
    trashDisabledTooltip:
      "GitOps mode is enabled. Manage software via your YAML files.",
  },
};

export const TrashDisabledObserver: Story = {
  args: {
    trashDisabled: true,
    trashDisabledTooltip:
      "Your role does not have permission to delete software.",
  },
};

export const ZeroInstallState: Story = {
  args: {
    installed: 0,
    pending: 0,
    failed: 0,
    labels: [],
  },
};
