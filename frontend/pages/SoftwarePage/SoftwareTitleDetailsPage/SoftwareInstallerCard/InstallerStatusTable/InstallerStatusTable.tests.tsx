import React from "react";
import { screen, within, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { getPathWithQueryParams } from "utilities/url";
import PATHS from "router/paths";

import InstallerStatusTable from "./InstallerStatusTable";

describe("InstallerStatusTable", () => {
  const render = createCustomRenderer();

  it("renders columns and links for statuses", () => {
    render(
      <InstallerStatusTable
        softwareId={123}
        teamId={5}
        status={{ installed: 0, pending: 1, failed: 3 }}
      />
    );

    // Check cell values (always "hosts", even for 1)
    const cells = screen.getAllByRole("cell");

    const installedLink = cells[0].querySelector("a.link-cell");
    const pendingLink = cells[1].querySelector("a.link-cell");
    const failedLink = cells[2].querySelector("a.link-cell");

    expect(installedLink).toHaveTextContent("0 hosts");
    expect(pendingLink).toHaveTextContent("1 host");
    expect(failedLink).toHaveTextContent("3 hosts");
  });

  it("renders correct header titles for install vs script package", () => {
    const { rerender } = render(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isScriptPackage={false}
      />
    );

    let headers = screen.getAllByRole("columnheader");
    expect(headers[0]).toHaveTextContent("Installed");
    expect(headers[1]).toHaveTextContent("Pending");
    expect(headers[2]).toHaveTextContent("Failed");

    rerender(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isScriptPackage
      />
    );

    headers = screen.getAllByRole("columnheader");
    expect(headers[0]).toHaveTextContent("Ran");
    expect(headers[1]).toHaveTextContent("Pending");
    expect(headers[2]).toHaveTextContent("Failed");
  });

  it("renders different tooltips for Android Play Store vs non-Android for pending", async () => {
    // non-Android: pending install/uninstall message
    const { user, rerender } = render(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp={false}
      />
    );

    let pendingHeader = screen.getByText(/pending/i);

    await user.hover(pendingHeader);

    await waitFor(() => {
      expect(
        screen.getByText(/Fleet is installing\/uninstalling or will/i)
      ).toBeInTheDocument();
    });

    // Android: Play Store–style message
    rerender(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp
      />
    );

    pendingHeader = screen.getByText(/pending/i);

    await user.hover(pendingHeader);

    await waitFor(() => {
      expect(
        screen.getByText(/Software will be installed or configuration will/i)
      ).toBeInTheDocument();
    });
  });

  it("hides installed tooltip for Android Play Store app", async () => {
    const { user, rerender } = render(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp={false}
      />
    );

    let installedHeader = screen.getByText(/installed/i);

    await user.hover(installedHeader);

    await waitFor(() => {
      expect(
        screen.getByText(/Software is installed on these hosts/i)
      ).toBeInTheDocument();
    });

    rerender(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp
      />
    );

    installedHeader = screen.getByText(/installed/i);

    await user.hover(installedHeader);

    // Installed tooltip returns null for Android Play Store
    await waitFor(() => {
      expect(
        screen.queryByText(/Software is installed on these hosts/i)
      ).not.toBeInTheDocument();
    });
  });

  it("renders failed tooltip text correctly for Android vs non-Android", async () => {
    const { user, rerender } = render(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp={false}
      />
    );

    let failedHeader = screen.getByText(/failed/i);

    await user.hover(failedHeader);

    await waitFor(() => {
      expect(
        screen.getByText(/These hosts failed to install\/uninstall software/i)
      ).toBeInTheDocument();
    });

    rerender(
      <InstallerStatusTable
        softwareId={1}
        teamId={1}
        status={{ installed: 0, pending: 0, failed: 0 }}
        isAndroidPlayStoreApp
      />
    );

    failedHeader = screen.getByText(/failed/i);

    await user.hover(failedHeader);

    await waitFor(() => {
      expect(
        screen.getByText(
          /Software failed to install or configuration failed to apply/i
        )
      ).toBeInTheDocument();
    });
  });
});
