import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { UserEvent } from "@testing-library/user-event";

import { renderWithSetup } from "test/test-utils";
import { ILabelSoftwareTitle } from "interfaces/label";
import paths from "router/paths";
import { stringToClipboard } from "utilities/copy_text";
import { getPathWithQueryParams } from "utilities/url";

import LibraryItemAccordion, {
  ILibraryItemAccordionProps,
} from "./LibraryItemAccordion";

jest.mock("utilities/copy_text", () => ({
  stringToClipboard: jest.fn(),
}));
const mockedStringToClipboard = stringToClipboard as jest.MockedFunction<
  typeof stringToClipboard
>;

const statusPath = (software_status: "installed" | "pending" | "failed") =>
  getPathWithQueryParams(paths.MANAGE_HOSTS, {
    software_title_id: 123,
    software_status,
    fleet_id: 0,
  });

const baseProps: ILibraryItemAccordionProps = {
  filename: "GoogleChrome.pkg",
  version: "149.0.7827.54",
  addedAt: new Date("2026-06-15T00:00:00Z").toISOString(),
  isActive: true,
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
};

const makeLabels = (count: number): ILabelSoftwareTitle[] =>
  Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: `Label ${i + 1}`,
  })) as ILabelSoftwareTitle[];

const renderAccordion = (overrides: Partial<ILibraryItemAccordionProps> = {}) =>
  renderWithSetup(<LibraryItemAccordion {...baseProps} {...overrides} />);

describe("LibraryItemAccordion", () => {
  describe("collapsed header", () => {
    it("renders the filename and version", () => {
      renderAccordion();
      expect(screen.getByText("GoogleChrome.pkg")).toBeVisible();
      expect(screen.getByText(/149\.0\.7827\.54/)).toBeVisible();
    });

    it("does not render the expanded panel by default", () => {
      renderAccordion();
      expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
      expect(screen.queryByText("Hash")).not.toBeInTheDocument();
    });
  });

  describe("expand / collapse", () => {
    it("expands when the header is clicked and collapses on a second click", async () => {
      const { user } = renderAccordion();

      const header = screen.getByRole("button", { expanded: false });
      await user.click(header);

      expect(screen.getByText("32 installed")).toBeVisible();
      expect(screen.getByText("5 pending")).toBeVisible();
      expect(screen.getByText("3 failed")).toBeVisible();
      expect(screen.getByText("Hash")).toBeVisible();

      await user.click(screen.getByRole("button", { expanded: true }));
      expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
    });
  });

  describe("badges", () => {
    it("renders the Latest badge when badgeState is 'latest'", () => {
      renderAccordion({ badgeState: "latest" });
      expect(screen.getByRole("button", { name: "Latest" })).toBeVisible();
    });

    it("renders the Pinned badge when badgeState is 'pinned'", () => {
      renderAccordion({ badgeState: "pinned" });
      expect(screen.getByRole("button", { name: "Pinned" })).toBeVisible();
    });

    it("renders the Major version badge when badgeState is 'majorVersion'", () => {
      renderAccordion({ badgeState: "majorVersion" });
      expect(
        screen.getByRole("button", { name: "Major version" })
      ).toBeVisible();
    });

    it("renders no badge when badgeState is undefined", () => {
      renderAccordion({ badgeState: undefined });
      expect(
        screen.queryByRole("button", { name: "Latest" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Pinned" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Major version" })
      ).not.toBeInTheDocument();
    });

    it("renders the label-count badge when labels are scoped", () => {
      renderAccordion({ badgeState: "latest", labels: makeLabels(7) });
      expect(screen.getByRole("button", { name: "7" })).toBeVisible();
    });

    it("renders 'All hosts' instead of the label-count when no labels are scoped", () => {
      renderAccordion({ badgeState: "latest", labels: [] });
      expect(screen.getByText("All hosts")).toBeVisible();
      expect(
        screen.queryByRole("button", { name: /^\d+$/ })
      ).not.toBeInTheDocument();
    });

    it("renders 'All hosts' when badgeState is 'majorVersion' with no scoped labels", () => {
      renderAccordion({ badgeState: "majorVersion", labels: [] });
      expect(screen.getByText("All hosts")).toBeVisible();
    });

    it("does not render 'All hosts' when badgeState is undefined (no badge means no fallback)", () => {
      renderAccordion({ badgeState: undefined, labels: [] });
      expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
    });

    it("renders a tooltip with the label list when hovering the count badge", async () => {
      const labels = [
        { id: 1, name: "Design" },
        { id: 2, name: "Engineering" },
        { id: 3, name: "IT" },
      ] as never;
      const { user } = renderAccordion({
        badgeState: "latest",
        labels,
        labelKind: "includeAll",
      });

      await user.hover(screen.getByRole("button", { name: "3" }));
      // Tooltip renders the heading inside a `<strong>` and the names as
      // sibling text nodes separated by `<br/>`. RTL can't match the
      // individual text nodes (they aren't elements), so assert against the
      // parent container's combined textContent — which preserves order but
      // strips the `<br/>` whitespace.
      await waitFor(() => {
        expect(screen.getByText("Include all:")).toBeInTheDocument();
      });
      const tooltipDiv =
        screen.getByText("Include all:").parentElement ?? document.body;
      expect(tooltipDiv).toHaveTextContent(/Design.*Engineering.*IT/);
    });

    it.each([
      ["latest", "Latest"],
      ["pinned", "Pinned"],
      ["majorVersion", "Major version"],
    ] as const)(
      "fires onBadgeClick when the %s badge is clicked",
      async (state, label) => {
        const onBadgeClick = jest.fn();
        const { user } = renderAccordion({ badgeState: state, onBadgeClick });

        await user.click(screen.getByRole("button", { name: label }));
        expect(onBadgeClick).toHaveBeenCalledTimes(1);
      }
    );

    it("does not propagate badge clicks to the header expand toggle", async () => {
      const onBadgeClick = jest.fn();
      const { user } = renderAccordion({ badgeState: "latest", onBadgeClick });

      // The header would expand if the click bubbled — verify it stays collapsed.
      await user.click(screen.getByRole("button", { name: "Latest" }));
      expect(onBadgeClick).toHaveBeenCalledTimes(1);
      expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
    });

    it("fires onLabelCountClick when the label-count badge is clicked", async () => {
      const onLabelCountClick = jest.fn();
      const { user } = renderAccordion({
        badgeState: "latest",
        labels: makeLabels(4),
        onLabelCountClick,
      });

      await user.click(screen.getByRole("button", { name: "4" }));
      expect(onLabelCountClick).toHaveBeenCalledTimes(1);
    });

    it("renders the label-count as static (non-button) when canEditSoftware is false", () => {
      renderAccordion({
        badgeState: "latest",
        labels: makeLabels(4),
        canEditSoftware: false,
      });
      expect(
        screen.queryByRole("button", { name: "4" })
      ).not.toBeInTheDocument();
      // The static span still displays the count.
      expect(screen.getByText("4")).toBeVisible();
    });

    it("fires onLabelCountClick when the 'All hosts' badge is clicked", async () => {
      const onLabelCountClick = jest.fn();
      const { user } = renderAccordion({
        badgeState: "latest",
        labels: [],
        onLabelCountClick,
      });

      // Exact name match — the outer header is also a `role="button"` whose
      // accessible name contains "All hosts" via descendant text.
      await user.click(screen.getByRole("button", { name: "All hosts" }));
      expect(onLabelCountClick).toHaveBeenCalledTimes(1);
    });

    it("renders 'All hosts' as static (non-button) when canEditSoftware is false", () => {
      renderAccordion({
        badgeState: "latest",
        labels: [],
        canEditSoftware: false,
      });
      expect(
        screen.queryByRole("button", { name: "All hosts" })
      ).not.toBeInTheDocument();
      // The static span still displays the label.
      expect(screen.getByText("All hosts")).toBeVisible();
    });
  });

  describe("inactive row", () => {
    it("hides all badges and the chevron interaction", async () => {
      const { user } = renderAccordion({
        isActive: false,
        badgeState: "latest",
        labels: makeLabels(3),
      });

      expect(
        screen.queryByRole("button", { name: "Latest" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "3" })
      ).not.toBeInTheDocument();

      await user.click(screen.getByRole("button"));
      expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
    });
  });

  describe("expanded panel — status counts", () => {
    it("renders zero-install state without crashing", async () => {
      const { user } = renderAccordion({
        installed: 0,
        pending: 0,
        failed: 0,
      });

      await user.click(screen.getByRole("button", { expanded: false }));

      expect(screen.getByText("0 installed")).toBeVisible();
      expect(screen.getByText("0 pending")).toBeVisible();
      expect(screen.getByText("0 failed")).toBeVisible();
    });

    it("renders status counts as links", async () => {
      const { user } = renderAccordion();

      await user.click(screen.getByRole("button", { expanded: false }));

      const installedLink = screen.getByRole("link", { name: /32 installed/ });
      expect(installedLink).toHaveAttribute("href", statusPath("installed"));
      expect(screen.getByRole("link", { name: /5 pending/ })).toHaveAttribute(
        "href",
        statusPath("pending")
      );
      expect(screen.getByRole("link", { name: /3 failed/ })).toHaveAttribute(
        "href",
        statusPath("failed")
      );
    });
  });

  describe("expanded panel — labels heading", () => {
    it("renders the 'Include any' heading by default", async () => {
      const { user } = renderAccordion({ labels: makeLabels(2) });
      await user.click(screen.getByRole("button", { expanded: false }));
      expect(screen.getByText("Include any")).toBeVisible();
    });

    it("renders the 'Include all' heading when labelKind is includeAll", async () => {
      const { user } = renderAccordion({
        labels: makeLabels(2),
        labelKind: "includeAll",
      });
      await user.click(screen.getByRole("button", { expanded: false }));
      expect(screen.getByText("Include all")).toBeVisible();
    });

    it("renders the 'Exclude any' heading when labelKind is excludeAny", async () => {
      const { user } = renderAccordion({
        labels: makeLabels(2),
        labelKind: "excludeAny",
      });
      await user.click(screen.getByRole("button", { expanded: false }));
      expect(screen.getByText("Exclude any")).toBeVisible();
    });
  });

  describe("expanded panel — hash copy", () => {
    it("copies the hash to the clipboard and shows a transient 'Copied!' message", async () => {
      mockedStringToClipboard.mockResolvedValueOnce(undefined);
      const { user } = renderAccordion();

      await user.click(screen.getByRole("button", { expanded: false }));
      await user.click(
        screen.getByRole("button", { name: "Copy hash to clipboard" })
      );

      expect(mockedStringToClipboard).toHaveBeenCalledWith(
        baseProps.hashSha256
      );
      expect(await screen.findByText("Copied!")).toBeVisible();
    });

    it("shows 'Copy failed' when the clipboard write rejects", async () => {
      mockedStringToClipboard.mockRejectedValueOnce(new Error("denied"));
      const { user } = renderAccordion();

      await user.click(screen.getByRole("button", { expanded: false }));
      await user.click(
        screen.getByRole("button", { name: "Copy hash to clipboard" })
      );

      expect(await screen.findByText("Copy failed")).toBeVisible();
    });
  });

  describe("download button", () => {
    it("fires onDownloadClick when clicked", async () => {
      const onDownloadClick = jest.fn();
      const { user } = renderAccordion({ onDownloadClick });

      await user.click(screen.getByRole("button", { expanded: false }));
      await user.click(
        screen.getByRole("button", { name: "Download installer" })
      );
      expect(onDownloadClick).toHaveBeenCalledTimes(1);
    });

    it("is omitted when canDownload is false", async () => {
      const { user } = renderAccordion({ canDownload: false });

      await user.click(screen.getByRole("button", { expanded: false }));
      expect(
        screen.queryByRole("button", { name: "Download installer" })
      ).not.toBeInTheDocument();
    });
  });

  describe("trash button", () => {
    it("is hidden entirely when canEditSoftware is false", async () => {
      const { user } = renderAccordion({ canEditSoftware: false });

      await user.click(screen.getByRole("button", { expanded: false }));

      expect(
        screen.queryByRole("button", { name: "Delete this version" })
      ).not.toBeInTheDocument();
      // Download stays — gated only by `canDownload`, not edit permission.
      expect(
        screen.getByRole("button", { name: "Download installer" })
      ).toBeVisible();
    });

    it("invokes onTrashClick when enabled", async () => {
      const onTrashClick = jest.fn();
      const { user } = renderAccordion({ onTrashClick });

      await user.click(screen.getByRole("button", { expanded: false }));
      await user.click(
        screen.getByRole("button", { name: "Delete this version" })
      );

      expect(onTrashClick).toHaveBeenCalledTimes(1);
    });
  });

  // Cross-cutting: every callback prop is optional, so a row with none wired
  // up must still click through every interactive element without throwing.
  describe("no-handler safety", () => {
    it("does not throw when interactive elements are clicked without handlers", async () => {
      const { user } = renderAccordion({
        badgeState: "latest",
        labels: makeLabels(2),
        // intentionally no onBadgeClick / onLabelCountClick / onDownloadClick /
        // onTrashClick — exercising the optional-callback no-op paths
      });

      await user.click(screen.getByRole("button", { name: "Latest" }));
      await user.click(screen.getByRole("button", { name: "2" }));
      await user.click(screen.getByRole("button", { expanded: false }));
      await user.click(
        screen.getByRole("button", { name: "Download installer" })
      );
      await user.click(
        screen.getByRole("button", { name: "Delete this version" })
      );
      // The test passes if none of the above throw.
    });
  });

  // The status-row label and the three icon tooltips switch between
  // package / script / Android Play Store wording. One test per mode: each
  // walks the full installed/pending/failed row so the whole presentation
  // for that mode is visible at a glance.
  describe("status row variants", () => {
    type StatusName = "installed" | "pending" | "failed";

    const STATUS_INDEX: Record<StatusName, number> = {
      installed: 0,
      pending: 1,
      failed: 2,
    };

    const getStatusCell = (container: HTMLElement, status: StatusName) =>
      container.querySelectorAll(".library-item-accordion__status-count")[
        STATUS_INDEX[status]
      ];

    const hoverStatusIcon = async (
      user: UserEvent,
      container: HTMLElement,
      status: StatusName
    ) => {
      const target = getStatusCell(container, status).querySelector(
        ".component__tooltip-wrapper__element"
      );
      if (!target) {
        throw new Error(`no tooltip wrapper found for status "${status}"`);
      }
      await user.hover(target);
    };

    const expectTooltipOnHover = async (
      user: UserEvent,
      container: HTMLElement,
      status: StatusName,
      expected: RegExp
    ) => {
      await hoverStatusIcon(user, container, status);
      expect(await screen.findByText(expected)).toBeInTheDocument();
    };

    // The info-outline icon trailing the "installed" count carries its own
    // tooltip whose copy varies by installer source (package / tarball /
    // Android). Hover it and assert the wording.
    const expectInfoTooltip = async (
      user: UserEvent,
      container: HTMLElement,
      expected: RegExp
    ) => {
      const wrapper = container.querySelector(
        ".library-item-accordion__status-counts-info .component__tooltip-wrapper__element"
      );
      if (!wrapper) {
        throw new Error("info-outline tooltip wrapper not found");
      }
      await user.hover(wrapper);
      expect(await screen.findByText(expected)).toBeInTheDocument();
    };

    it("renders the installed/pending/failed labels and tooltips for a package", async () => {
      const { user, container } = renderAccordion();
      await user.click(screen.getByRole("button", { expanded: false }));

      expect(screen.getByText("32 installed")).toBeVisible();
      expect(screen.getByText("5 pending")).toBeVisible();
      expect(screen.getByText("3 failed")).toBeVisible();

      await expectTooltipOnHover(
        user,
        container,
        "installed",
        /Software is installed on these hosts/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "pending",
        /Fleet is installing\/uninstalling/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "failed",
        /failed to install\/uninstall/i
      );

      // Info-outline tooltip on the installed count: default (package) wording
      // includes all three sources — policy automation, setup experience, and
      // manual install.
      await expectInfoTooltip(
        user,
        container,
        /policy automation.*setup experience.*manual install/i
      );
    });

    it("renders the installed/pending/failed labels and tooltips for a tarball package", async () => {
      const { user, container } = renderAccordion({ isTarballPackage: true });
      await user.click(screen.getByRole("button", { expanded: false }));

      // Tarballs don't swap labels or per-status icon tooltips — only the
      // info-outline tooltip changes (no setup-experience leg).
      expect(screen.getByText("32 installed")).toBeVisible();
      expect(screen.getByText("5 pending")).toBeVisible();
      expect(screen.getByText("3 failed")).toBeVisible();

      await expectTooltipOnHover(
        user,
        container,
        "installed",
        /Software is installed on these hosts/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "pending",
        /Fleet is installing\/uninstalling/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "failed",
        /failed to install\/uninstall/i
      );

      await expectInfoTooltip(
        user,
        container,
        /policy automation or manual install/i
      );
      // Setup-experience leg must be absent for tarballs.
      expect(screen.queryByText(/setup experience/i)).not.toBeInTheDocument();
    });

    it("renders the installed/pending/failed labels and tooltips for a script-only package", async () => {
      const { user, container } = renderAccordion({ isScriptPackage: true });
      await user.click(screen.getByRole("button", { expanded: false }));

      // Script-only swaps "installed" → "ran"; pending/failed labels are unchanged.
      expect(screen.getByText("32 ran")).toBeVisible();
      expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
      expect(screen.getByText("5 pending")).toBeVisible();
      expect(screen.getByText("3 failed")).toBeVisible();

      await expectTooltipOnHover(
        user,
        container,
        "installed",
        /script successfully/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "pending",
        /Fleet is running the script/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "failed",
        /failed to run the script/i
      );
    });

    it("drops the policy-automation leg from the info tooltip for iOS/iPadOS apps", async () => {
      const { user, container } = renderAccordion({ isIosOrIpadosApp: true });
      await user.click(screen.getByRole("button", { expanded: false }));

      await expectInfoTooltip(
        user,
        container,
        /setup experience or manual install/i
      );
      // Policy automation is macOS-only on Apple VPP — make sure that
      // leg is gone from the iOS/iPadOS copy.
      expect(screen.queryByText(/policy automation/i)).not.toBeInTheDocument();
    });

    it("renders the installed/pending/failed labels and tooltips for an Android Play Store app", async () => {
      const { user, container } = renderAccordion({
        androidPlayStoreId: "com.example.app",
      });
      await user.click(screen.getByRole("button", { expanded: false }));

      // Android does NOT swap the installed label (only script does).
      expect(screen.getByText("32 installed")).toBeVisible();
      expect(screen.getByText("5 pending")).toBeVisible();
      expect(screen.getByText("3 failed")).toBeVisible();

      // The installed icon has no tooltip on Android — assert structurally
      // since there's nothing to hover. Package/script modes wrap the success
      // icon in a TooltipWrapper (`.component__tooltip-wrapper` is the cell's
      // first child); Android leaves the bare Icon there.
      const installedCell = getStatusCell(container, "installed");
      expect(
        installedCell?.firstElementChild?.classList.contains(
          "component__tooltip-wrapper"
        )
      ).toBe(false);

      await expectTooltipOnHover(
        user,
        container,
        "pending",
        /next time the host checks in/i
      );
      await expectTooltipOnHover(
        user,
        container,
        "failed",
        /configuration failed to apply/i
      );

      // Info-outline tooltip collapses to a Play Store one-liner on Android.
      await expectInfoTooltip(
        user,
        container,
        /latest status from the Google Play Store/i
      );
    });
  });
});
