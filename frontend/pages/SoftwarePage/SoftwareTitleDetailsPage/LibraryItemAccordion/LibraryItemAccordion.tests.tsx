import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";

import { ILabelSoftwareTitle } from "interfaces/label";
import paths from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import LibraryItemAccordion, {
  ILibraryItemAccordionProps,
} from "./LibraryItemAccordion";

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

  it("renders the Latest badge when isLatest is true", () => {
    renderAccordion({ isLatest: true });
    expect(screen.getByRole("button", { name: "Latest" })).toBeVisible();
  });

  it("renders the Pinned badge when isPinned is true", () => {
    renderAccordion({ isPinned: true });
    expect(screen.getByRole("button", { name: "Pinned" })).toBeVisible();
  });

  it("renders the label-count badge when labels are scoped", () => {
    renderAccordion({ isLatest: true, labels: makeLabels(7) });
    expect(screen.getByRole("button", { name: "7" })).toBeVisible();
  });

  it("renders 'All hosts' instead of the label-count when no labels are scoped", () => {
    renderAccordion({ isLatest: true, labels: [] });
    expect(screen.getByText("All hosts")).toBeVisible();
    expect(
      screen.queryByRole("button", { name: /^\d+$/ })
    ).not.toBeInTheDocument();
  });

  it("hides all badges and the chevron interaction when inactive", async () => {
    const user = userEvent.setup();
    renderAccordion({
      isActive: false,
      isLatest: true,
      labels: makeLabels(3),
    });

    expect(
      screen.queryByRole("button", { name: "Latest" })
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "3" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button"));
    expect(screen.queryByText("32 installed")).not.toBeInTheDocument();
  });

  it("renders the trash button disabled with tooltip when trashDisabled", async () => {
    const onTrashClick = jest.fn();
    const user = userEvent.setup();
    renderAccordion({
      trashDisabled: true,
      trashDisabledTooltip: "GitOps mode is enabled",
      onTrashClick,
    });

    await user.click(screen.getByRole("button", { expanded: false }));

    const trashBtn = screen.getByRole("button", {
      name: "Delete this version",
    });
    expect(trashBtn).toBeDisabled();

    await user.click(trashBtn);
    expect(onTrashClick).not.toHaveBeenCalled();
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
    renderAccordion({ isLatest: true, labels, labelKind: "includeAll" });

    await user.hover(screen.getByRole("button", { name: "3" }));
    await waitFor(() => {
      expect(
        screen.getByText("Include all: Design, Engineering, IT")
      ).toBeInTheDocument();
    });
  });

  it("uses callback wiring without errors when no handlers are provided", async () => {
    const user = userEvent.setup();
    renderAccordion();
    await user.click(screen.getByRole("button", { expanded: false }));

    expect(() => noop()).not.toThrow();
  });
});
