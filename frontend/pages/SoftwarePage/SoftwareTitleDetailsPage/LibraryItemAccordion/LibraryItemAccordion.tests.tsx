import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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
  installed: 32,
  pending: 5,
  failed: 3,
  installedPath: statusPath("installed"),
  pendingPath: statusPath("pending"),
  failedPath: statusPath("failed"),
  hashSha256:
    "af001543fcc5fbf484203b207d8af4fce44fc6975ca3db0eac49a49581af29b7",
  downloadUrl: "https://example.com/installer.pkg",
};

const makeLabels = (count: number): ILabelSoftwareTitle[] =>
  Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: `Label ${i + 1}`,
  })) as ILabelSoftwareTitle[];

const renderAccordion = (overrides: Partial<ILibraryItemAccordionProps> = {}) =>
  render(<LibraryItemAccordion {...baseProps} {...overrides} />);

describe("LibraryItemAccordion", () => {
  it("renders filename and version line in the collapsed header", () => {
    renderAccordion();
    expect(screen.getByText("GoogleChrome.pkg")).toBeVisible();
    expect(screen.getByText(/149\.0\.7827\.54/)).toBeVisible();
  });

  it("starts collapsed and does not render the expanded panel", () => {
    renderAccordion();
    expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
    expect(screen.queryByText("Hash")).not.toBeInTheDocument();
  });

  it("expands when the header is clicked and collapses on a second click", async () => {
    const user = userEvent.setup();
    renderAccordion();

    const header = screen.getByRole("button", { expanded: false });
    await user.click(header);

    expect(screen.getByText("32 installed")).toBeVisible();
    expect(screen.getByText("5 pending")).toBeVisible();
    expect(screen.getByText("3 failed")).toBeVisible();
    expect(screen.getByText("Hash")).toBeVisible();

    await user.click(screen.getByRole("button", { expanded: true }));
    expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
  });

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
    expect(screen.getByRole("button", { name: "Major version" })).toBeVisible();
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

  it("hides all badges and the chevron interaction when inactive", async () => {
    const user = userEvent.setup();
    renderAccordion({
      isActive: false,
      badgeState: "latest",
      labels: makeLabels(3),
    });

    expect(
      screen.queryByRole("button", { name: "Latest" })
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "3" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button"));
    expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
  });

  it("hides the trash button entirely when canEditSoftware is false", async () => {
    const user = userEvent.setup();
    renderAccordion({ canEditSoftware: false });

    await user.click(screen.getByRole("button", { expanded: false }));

    expect(
      screen.queryByRole("button", { name: "Delete this version" })
    ).not.toBeInTheDocument();
    // Download stays — gated only by `downloadUrl`, not edit permission.
    expect(
      screen.getByRole("button", { name: "Download installer" })
    ).toBeVisible();
  });

  it("invokes onTrashClick when enabled", async () => {
    const onTrashClick = jest.fn();
    const user = userEvent.setup();
    renderAccordion({ onTrashClick });

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(
      screen.getByRole("button", { name: "Delete this version" })
    );

    expect(onTrashClick).toHaveBeenCalledTimes(1);
  });

  it("renders zero-install state without crashing", async () => {
    const user = userEvent.setup();
    renderAccordion({ installed: 0, pending: 0, failed: 0 });

    await user.click(screen.getByRole("button", { expanded: false }));

    expect(screen.getByText("0 installed")).toBeVisible();
    expect(screen.getByText("0 pending")).toBeVisible();
    expect(screen.getByText("0 failed")).toBeVisible();
  });

  it("renders the 'Include any' heading by default", async () => {
    const user = userEvent.setup();
    renderAccordion({ labels: makeLabels(2) });
    await user.click(screen.getByRole("button", { expanded: false }));
    expect(screen.getByText("Include any")).toBeVisible();
  });

  it("renders the 'Include all' heading when labelKind is includeAll", async () => {
    const user = userEvent.setup();
    renderAccordion({ labels: makeLabels(2), labelKind: "includeAll" });
    await user.click(screen.getByRole("button", { expanded: false }));
    expect(screen.getByText("Include all")).toBeVisible();
  });

  it("renders the 'Exclude any' heading when labelKind is excludeAny", async () => {
    const user = userEvent.setup();
    renderAccordion({ labels: makeLabels(2), labelKind: "excludeAny" });
    await user.click(screen.getByRole("button", { expanded: false }));
    expect(screen.getByText("Exclude any")).toBeVisible();
  });

  it("renders a tooltip with the label list when hovering the count badge", async () => {
    const user = userEvent.setup();
    const labels = [
      { id: 1, name: "Design" },
      { id: 2, name: "Engineering" },
      { id: 3, name: "IT" },
    ] as never;
    renderAccordion({ badgeState: "latest", labels, labelKind: "includeAll" });

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
      const user = userEvent.setup();
      renderAccordion({ badgeState: state, onBadgeClick });

      await user.click(screen.getByRole("button", { name: label }));
      expect(onBadgeClick).toHaveBeenCalledTimes(1);
    }
  );

  it("does not propagate badge clicks to the header expand toggle", async () => {
    const onBadgeClick = jest.fn();
    const user = userEvent.setup();
    renderAccordion({ badgeState: "latest", onBadgeClick });

    // The header would expand if the click bubbled — verify it stays collapsed.
    await user.click(screen.getByRole("button", { name: "Latest" }));
    expect(onBadgeClick).toHaveBeenCalledTimes(1);
    expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
  });

  it("fires onLabelCountClick when the label-count badge is clicked", async () => {
    const onLabelCountClick = jest.fn();
    const user = userEvent.setup();
    renderAccordion({
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
    expect(screen.queryByRole("button", { name: "4" })).not.toBeInTheDocument();
    // The static span still displays the count.
    expect(screen.getByText("4")).toBeVisible();
  });

  it("fires onDownloadClick when the download button is clicked", async () => {
    const onDownloadClick = jest.fn();
    const user = userEvent.setup();
    renderAccordion({ onDownloadClick });

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(
      screen.getByRole("button", { name: "Download installer" })
    );
    expect(onDownloadClick).toHaveBeenCalledTimes(1);
  });

  it("omits the download button when downloadUrl is not provided", async () => {
    const user = userEvent.setup();
    renderAccordion({ downloadUrl: undefined });

    await user.click(screen.getByRole("button", { expanded: false }));
    expect(
      screen.queryByRole("button", { name: "Download installer" })
    ).not.toBeInTheDocument();
  });

  it("renders status counts as links when expanded", async () => {
    const user = userEvent.setup();
    renderAccordion();

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

  it("copies the hash to the clipboard and shows a transient 'Copied!' message", async () => {
    mockedStringToClipboard.mockResolvedValueOnce(undefined);
    const user = userEvent.setup();
    renderAccordion();

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(
      screen.getByRole("button", { name: "Copy hash to clipboard" })
    );

    expect(mockedStringToClipboard).toHaveBeenCalledWith(baseProps.hashSha256);
    expect(await screen.findByText("Copied!")).toBeVisible();
  });

  it("shows 'Copy failed' when the clipboard write rejects", async () => {
    mockedStringToClipboard.mockRejectedValueOnce(new Error("denied"));
    const user = userEvent.setup();
    renderAccordion();

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(
      screen.getByRole("button", { name: "Copy hash to clipboard" })
    );

    expect(await screen.findByText("Copy failed")).toBeVisible();
  });

  it("does not throw when the row is rendered with no handlers and interactive elements are clicked", async () => {
    const user = userEvent.setup();
    renderAccordion({
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
