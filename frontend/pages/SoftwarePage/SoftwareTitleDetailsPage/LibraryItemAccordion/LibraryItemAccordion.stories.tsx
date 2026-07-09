import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";

import { ILabelSoftwareTitle } from "interfaces/label";
import paths from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import LibraryItemAccordion from "./LibraryItemAccordion";

// Needed because the embedded `SoftwareIcon` (rendered for installerType
// "app-store") uses `useQuery` internally. Without a QueryClientProvider in
// scope, switching the `installerType` control to "app-store" throws.
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
type CustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: React.FC<CustomQueryClientProviderProps> = QueryClientProvider;

const labels7: ILabelSoftwareTitle[] = Array.from({ length: 7 }, (_, i) => ({
  id: i + 1,
  name: `Label ${i + 1}`,
})) as ILabelSoftwareTitle[];

const statusPath = (software_status: "installed" | "pending" | "failed") =>
  getPathWithQueryParams(paths.MANAGE_HOSTS, {
    software_title_id: 123,
    software_status,
    fleet_id: 0,
  });

const meta: Meta<typeof LibraryItemAccordion> = {
  title: "Pages/SoftwareTitleDetailsPage/LibraryItemAccordion",
  component: LibraryItemAccordion,
  args: {
    filename: "GoogleChrome.pkg",
    version: "149.0.7827.54",
    addedAt: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(),
    isActive: true,
    badgeState: "latest",
    labels: labels7,
    canEditSoftware: true,
    installed: 32,
    pending: 5,
    failed: 3,
    installedPath: statusPath("installed"),
    pendingPath: statusPath("pending"),
    failedPath: statusPath("failed"),
    hashSha256:
      "af001543fcc5fbf484203b207d8af4fce44fc6975ca3db0eac49a49581af29b7",
    canDownload: true,
  },
  decorators: [
    (Story) => (
      <CustomQueryClientProvider client={queryClient}>
        <Story />
      </CustomQueryClientProvider>
    ),
  ],
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

/** FMA row: the pin is a clickable button that opens the versions modal. */
export const LatestActive: Story = {
  args: {
    badgeState: "latest",
    onBadgeClick: () => undefined,
  },
};

export const PinnedActive: Story = {
  args: {
    badgeState: "pinned",
    onBadgeClick: () => undefined,
  },
};

export const MajorVersionPinnedActive: Story = {
  args: {
    badgeState: "majorVersion",
    onBadgeClick: () => undefined,
  },
};

export const AllHostsNoLabels: Story = {
  args: {
    badgeState: "latest",
    labels: [],
  },
};

export const Inactive: Story = {
  args: {
    isActive: false,
    badgeState: undefined,
    labels: [],
    version: "148.0.7778.179",
    addedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 20).toISOString(),
  },
};

/** Active row, user lacks edit permission. The label-count badge demotes to a
 * static span (no button + no click handler), the expanded-panel labels list
 * renders as plain text rather than a CustomLink, and the trash button is
 * hidden entirely. The download button stays (gated only by `canDownload`). */
export const ActiveCannotEditSoftware: Story = {
  args: {
    canEditSoftware: false,
    badgeState: "latest",
    labels: labels7,
  },
};

/** Inactive row, user lacks edit permission. The "Select Actions > Versions
 * and pin this version to rollback" hover tooltip is suppressed because the
 * user can't reach that menu anyway. */
export const InactiveCannotEditSoftware: Story = {
  args: {
    canEditSoftware: false,
    isActive: false,
    badgeState: undefined,
    labels: [],
    version: "148.0.7778.179",
    addedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 20).toISOString(),
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
