/**
 * Multi-row list stories. Single-row prop variants live in
 * `LibraryItemAccordion.stories.tsx`.
 *
 * Two pieces of indirection:
 *   - `<StoryRow>` injects path props + a `canEditSoftware: true` default so
 *     rows stay terse.
 *   - `LibraryItemAccordionListDemo` clones each child to inject `labels` /
 *     `labelKind` / `badgeState` from the controls panel.
 *
 * New `<LibraryItemAccordion>` props may need wiring through one of these
 * before they surface in any story here.
 */

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

import LibraryItemAccordion, {
  ILibraryItemAccordionProps,
  LibraryItemLabelKind,
} from "./LibraryItemAccordion";
import LibraryItemAccordionList from "./LibraryItemAccordionList";

const daysAgo = (n: number) =>
  new Date(Date.now() - 1000 * 60 * 60 * 24 * n).toISOString();

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

// React Query v3 with React 18 needs `children` explicitly typed on the
// provider. Mirrors the pattern in frontend/router/index.tsx and other stories.
type CustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: React.FC<CustomQueryClientProviderProps> = QueryClientProvider;

const FAKE_LABEL_NAMES = [
  "Engineering",
  "Design",
  "Marketing",
  "Sales",
  "Customer success",
  "Finance",
  "Legal",
  "IT",
  "Workstations",
  "Servers",
  "macOS workstations",
  "Windows workstations",
  "Linux servers",
  "Production",
  "Staging",
];

const generateLabels = (count: number): ILabelSoftwareTitle[] =>
  Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: FAKE_LABEL_NAMES[i] ?? `Label ${i + 1}`,
  })) as ILabelSoftwareTitle[];

// Exposes labelKind/labelCount as Storybook args; builds the label list,
// wraps rows in LibraryItemAccordionList, and clones each row to inject the
// labels so story authors don't repeat them on every accordion.
type BadgeState = "latest" | "pinned" | "majorVersion";

interface ILibraryItemAccordionListDemoProps {
  /** Label scope applied to every row — drives the header label-count badge
   * tooltip and the expanded "Include any / Include all / Exclude any"
   * heading. */
  labelKind: LibraryItemLabelKind;
  /** Number of fake labels assigned to every row. 0 = no scoped labels
   * (falls back to the "All hosts" badge on active rows). */
  labelCount: number;
  /** Which badge the active row(s) display. Injected into every active
   * accordion child via `cloneElement` — each story doesn't need to set
   * `badgeState` itself. `pinned` → "Pinned" badge (pin icon). `majorVersion`
   * → "Major version" badge (same pin icon, distinct label). */
  badgeState: BadgeState;
  children?: React.ReactNode;
}

// Recursively unwraps React.Fragment so stories can use `<>...</>`. Neither
// `Children.map` nor `Children.toArray` traverses fragments — they treat
// them as single leaf elements — which would route cloneElement's injected
// props to the fragment wrapper instead of the accordion rows.
const flattenFragments = (nodes: React.ReactNode): React.ReactElement[] => {
  const out: React.ReactElement[] = [];
  React.Children.forEach(nodes, (child) => {
    if (!React.isValidElement(child)) return;
    if (child.type === React.Fragment) {
      out.push(
        ...flattenFragments(
          (child.props as { children?: React.ReactNode }).children
        )
      );
    } else {
      out.push(child);
    }
  });
  return out;
};

// Each control-panel option maps directly to one `badgeState` value — no
// boolean toggling, the prop is its own discriminated union now.
const badgeOverridesFor = (state: BadgeState) => ({ badgeState: state });

// Stub URLs from the production utility so install-status counts render as
// CustomLinks with realistic-looking hrefs. Only the shape matters here.
const statusPath = (software_status: "installed" | "pending" | "failed") =>
  getPathWithQueryParams(paths.MANAGE_HOSTS, {
    software_title_id: 123,
    software_status,
    fleet_id: 0,
  });

const STORYBOOK_PATHS = {
  installedPath: statusPath("installed"),
  pendingPath: statusPath("pending"),
  failedPath: statusPath("failed"),
};

// Shim around `<LibraryItemAccordion>`: injects path props + a
// `canEditSoftware: true` default so non-permission stories stay terse.
// Permission stories override `canEditSoftware` explicitly. The Demo
// wrapper still injects labels/labelKind/badgeState via cloneElement.
type IStoryRowProps = Omit<
  ILibraryItemAccordionProps,
  "installedPath" | "pendingPath" | "failedPath" | "canEditSoftware"
> & { canEditSoftware?: boolean };
const StoryRow = ({ canEditSoftware = true, ...props }: IStoryRowProps) => (
  <LibraryItemAccordion
    {...props}
    {...STORYBOOK_PATHS}
    canEditSoftware={canEditSoftware}
  />
);

const LibraryItemAccordionListDemo = ({
  labelKind,
  labelCount,
  badgeState,
  children,
}: ILibraryItemAccordionListDemoProps) => {
  const labels = generateLabels(labelCount);
  const rows = flattenFragments(children);
  return (
    <LibraryItemAccordionList>
      {rows.map((child, i) => {
        const childProps = child.props as ILibraryItemAccordionProps;
        // Only push a badge override onto active rows — inactive rows hide all
        // badges, so the override would be a no-op but it keeps the cloned
        // props sane.
        const badgeProps = childProps.isActive
          ? badgeOverridesFor(badgeState)
          : {};
        return React.cloneElement(
          child as React.ReactElement<ILibraryItemAccordionProps>,
          {
            labels,
            labelKind,
            ...badgeProps,
            key: child.key ?? i,
          }
        );
      })}
    </LibraryItemAccordionList>
  );
};

const meta: Meta<typeof LibraryItemAccordionListDemo> = {
  title: "Pages/SoftwareTitleDetailsPage/LibraryItemAccordionList",
  component: LibraryItemAccordionListDemo,
  args: {
    labelKind: "includeAny",
    labelCount: 0,
    badgeState: "latest",
  },
  argTypes: {
    labelKind: {
      control: "select",
      options: ["includeAny", "includeAll", "excludeAny"],
      description:
        "Label scope applied to every row in the story (drives the badge tooltip + expanded heading).",
    },
    labelCount: {
      control: { type: "number", min: 0, max: FAKE_LABEL_NAMES.length },
      description:
        "Number of fake labels assigned to every row. 0 = no scoped labels (falls back to 'All hosts').",
    },
    badgeState: {
      control: "select",
      options: ["latest", "pinned", "majorVersion"],
      description:
        "Badge shown on active rows. 'latest' → 'Latest' (refresh icon). 'pinned' → 'Pinned' (pin icon). 'majorVersion' → 'Major version' (pin icon).",
    },
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

type Story = StoryObj<typeof LibraryItemAccordionListDemo>;

// Thin shim so each story can stay terse: hands the current args to the
// wrapper component, which builds labels and clones them onto every row.
const renderList = (
  args: ILibraryItemAccordionListDemoProps,
  rows: React.ReactNode
) => (
  <LibraryItemAccordionListDemo {...args}>{rows}</LibraryItemAccordionListDemo>
);

/** Scenario: user pinned the FMA to **major version 149** via Actions > Versions.
 * The `Pinned` badge sits on the latest cached 149.x release (the row that
 * satisfies the major); older 149.x patches and the previous major (148.x) are
 * rendered inactive. The badge label itself is the same as for an exact-version
 * pin — the distinction is data-driven (and surfaced in the Versions modal /
 * activity feed `pinned_version: "^149"`), not visual at the row level. */
export const PinnedToMajorVersion: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="Google Chrome"
          version="149.0.7827.54"
          addedAt={daysAgo(1)}
          installerType="package"
          isFma
          isLatestFmaVersion
          isActive
          installed={32}
          pending={5}
          failed={3}
          hashSha256="af001543fcc5fbf484203b207d8af4fce44fc6975ca3db0eac49a49581af29b7"
          downloadUrl="https://example.com/chrome-149.0.7827.54.pkg"
        />
        <StoryRow
          filename="Google Chrome"
          version="149.0.7800.10"
          addedAt={daysAgo(10)}
          installerType="package"
          isFma
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="Google Chrome"
          version="148.0.7778.179"
          addedAt={daysAgo(28)}
          installerType="package"
          isFma
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

// Note: there is intentionally no "MixedInstallerTypes" story. A software
// title binds to one installer path (FMA, custom package, VPP App Store, or
// Google Play — mutually exclusive in the schema), so the production page
// will never render different installer types in the same list. The per-type
// row treatments are still visible across the remaining stories
// (`AndroidFmaSingleVersion`, `AppStoreVppSingleVersion`, the custom-package
// variants, and the doc-only Windows/macOS mixed custom+FMA stories below).

/** Single cached version of a Google Play FMA (Chrome for Android). Android
 * Play Store apps don't cache multiple versions — the version chip always
 * reads "Latest" via `AndroidLatestVersionWithTooltip`, since the version is
 * pulled live from the Play Store rather than tracked per-row. The list will
 * therefore only ever contain one row for an Android FMA. */
export const AndroidFmaSingleVersion: Story = {
  render: (args) =>
    renderList(
      args,
      <StoryRow
        filename="Google Chrome"
        addedAt={daysAgo(1)}
        installerType="app-store"
        androidPlayStoreId="com.android.chrome"
        isActive
        installed={18}
        pending={2}
        failed={1}
      />
    ),
};

/** Three cached versions of one **custom macOS package** (`.pkg`). Same title,
 * different versions — the realistic "user uploaded patches over time" view.
 * The newest row is active+latest with install counts; the older two are
 * inactive (greyed) and show the rollback hover tooltip. */
export const MacCustomPackageMultipleVersions: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="AcmeHelper.pkg"
          version="2.4.0"
          addedAt={daysAgo(2)}
          installerType="package"
          isActive
          installed={47}
          pending={3}
          failed={1}
          hashSha256="b9d3a9d6c1e9442f9c0bb56af4f37b87f0bcb6df7f8db5a30e1bdce20c40a8d3"
          downloadUrl="https://example.com/acme-helper-2.4.0.pkg"
        />
        <StoryRow
          filename="AcmeHelper.pkg"
          version="2.3.5"
          addedAt={daysAgo(18)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="AcmeHelper.pkg"
          version="2.2.0"
          addedAt={daysAgo(60)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

/** Three cached versions of one **custom Windows package** (`.msi`). At the
 * component level Windows custom packages render the same as macOS (file-pkg
 * icon, "Custom package" label is hidden by `hideInstallerType`) — only the
 * filename extension hints at the OS. */
export const WindowsCustomPackageMultipleVersions: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="NotepadPlusPlus.msi"
          version="8.6.9"
          addedAt={daysAgo(3)}
          installerType="package"
          isActive
          installed={28}
          pending={4}
          failed={2}
          hashSha256="2e8a4f3b9c1d5e7a8b6c2f0d1e3a5b7c9d2e4f6a8b0c1d3e5f7a9b1c3d5e7f9a"
          downloadUrl="https://example.com/npp-8.6.9.msi"
        />
        <StoryRow
          filename="NotepadPlusPlus.msi"
          version="8.6.4"
          addedAt={daysAgo(22)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="NotepadPlusPlus.msi"
          version="8.5.8"
          addedAt={daysAgo(70)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

/** **Documentation only — cannot occur in production.** A single title has
 * one installer path (FMA or custom), so an FMA and a custom package never
 * appear in the same list. This story stacks an FMA Windows row against a
 * custom Windows `.msi` so designers can verify the FMA row's "(latest)"
 * suffix and "Actions > Edit" tooltip on the version chip — the only visual
 * cue separating it from a custom Windows row (both share the `file-pkg`
 * icon). */
export const WindowsMixedCustomAndFma: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="Mozilla Firefox"
          version="131.0.3"
          addedAt={daysAgo(1)}
          installerType="package"
          isFma
          isLatestFmaVersion
          isActive
          installed={54}
          pending={6}
          failed={2}
          hashSha256="9f2c4e6a8b0d1f3e5a7c9b1d3f5e7a9c1b3d5f7e9a1c3b5d7f9e1a3c5b7d9f1e"
          downloadUrl="https://example.com/firefox-131.0.3.msi"
        />
        <StoryRow
          filename="Mozilla Firefox"
          version="130.0.1"
          addedAt={daysAgo(20)}
          installerType="package"
          isFma
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="CompanyVPN.msi"
          version="4.1.0"
          addedAt={daysAgo(7)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
          hashSha256="3a7c9e1b5d2f4a6c8e0b2d4f6a8c0e2b4d6f8a0c2e4b6d8f0a2c4e6b8d0f2a4c"
        />
        <StoryRow
          filename="CompanyVPN.msi"
          version="4.0.2"
          addedAt={daysAgo(40)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

/** **Documentation only — cannot occur in production.** Mac mirror of
 * `WindowsMixedCustomAndFma`. An FMA macOS `.pkg` stacked against a custom
 * macOS `.pkg` so the FMA row's "(latest)" suffix and "Actions > Edit"
 * tooltip can be compared against a plain custom-package row. */
export const MacOSMixedCustomAndFma: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="Slack"
          version="4.39.95"
          addedAt={daysAgo(1)}
          installerType="package"
          isFma
          isLatestFmaVersion
          isActive
          installed={87}
          pending={4}
          failed={2}
          hashSha256="d4e7a1c3b5f9e1a3c5b7d9f1e3a5c7b9d1f3e5a7c9b1d3f5e7a9c1b3d5f7e9a1"
          downloadUrl="https://example.com/slack-4.39.95.pkg"
        />
        <StoryRow
          filename="Slack"
          version="4.38.121"
          addedAt={daysAgo(25)}
          installerType="package"
          isFma
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="DesignTool.pkg"
          version="3.2.1"
          addedAt={daysAgo(8)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
          hashSha256="7e3a9c1b5d2f4a6c8e0b2d4f6a8c0e2b4d6f8a0c2e4b6d8f0a2c4e6b8d0f2a4e"
        />
        <StoryRow
          filename="DesignTool.pkg"
          version="3.1.0"
          addedAt={daysAgo(50)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

/** Single cached version of an Apple App Store (VPP) app. Like Google Play
 * apps, VPP apps don't cache multiple versions — the `version` value updates
 * hourly from the App Store (note the "Updated every hour" tooltip on the
 * version chip), so the list will only ever contain one row. */
export const AppStoreVppSingleVersion: Story = {
  render: (args) =>
    renderList(
      args,
      <StoryRow
        filename="1Password 7 - Password Manager"
        version="7.9.11"
        addedAt={daysAgo(2)}
        installerType="app-store"
        isActive
        installed={42}
        pending={3}
        failed={0}
      />
    ),
};

/** Three cached versions of an **iOS/iPadOS in-house `.ipa`** uploaded as a
 * custom software installer (enterprise app distributed outside the App
 * Store). At the component level this renders the same way as any other
 * custom package — `file-pkg` icon, "Custom package" label hidden by
 * `hideInstallerType`, only the `.ipa` filename extension hints at iOS. Note:
 * the iOS-specific managed-app configuration plist (the `configuration` field
 * on `ISoftwarePackage`) is rendered elsewhere on the page, not in the
 * accordion. */
export const IOSInHouseIpaMultipleVersions: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="AcmeWarehouse.ipa"
          version="5.2.1"
          addedAt={daysAgo(4)}
          installerType="package"
          isActive
          installed={36}
          pending={2}
          failed={1}
          hashSha256="6b1d3a5c7e9f0b2d4a6c8e1f3b5d7a9c0e2f4b6d8a0c1e3f5b7d9a1c3e5f7b9d"
          downloadUrl="https://example.com/acme-warehouse-5.2.1.ipa"
        />
        <StoryRow
          filename="AcmeWarehouse.ipa"
          version="5.1.0"
          addedAt={daysAgo(28)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
        <StoryRow
          filename="AcmeWarehouse.ipa"
          version="5.0.3"
          addedAt={daysAgo(75)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
        />
      </>
    ),
};

/** **Documentation only — cannot occur in production.** A single software
 * title binds to one installer path (VPP App Store **or** in-house `.ipa`,
 * not both). This story stacks an iOS VPP app row against an in-house `.ipa`
 * row so designers can compare the two side-by-side: the Apple App Store
 * icon + "Updated every hour" version tooltip vs the `file-pkg` icon + plain
 * version chip. */
export const IOSMixedVppAndInHouseIpa: Story = {
  render: (args) =>
    renderList(
      args,
      <>
        <StoryRow
          filename="Microsoft Authenticator"
          version="6.8.14"
          addedAt={daysAgo(2)}
          installerType="app-store"
          isActive
          installed={64}
          pending={5}
          failed={1}
        />
        <StoryRow
          filename="AcmeWarehouse.ipa"
          version="5.2.1"
          addedAt={daysAgo(6)}
          installerType="package"
          isActive={false}
          installed={0}
          pending={0}
          failed={0}
          hashSha256="6b1d3a5c7e9f0b2d4a6c8e1f3b5d7a9c0e2f4b6d8a0c1e3f5b7d9a1c3e5f7b9d"
        />
      </>
    ),
};
