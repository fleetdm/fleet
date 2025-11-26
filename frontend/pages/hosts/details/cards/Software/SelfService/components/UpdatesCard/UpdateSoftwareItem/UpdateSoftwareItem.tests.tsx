import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import { noop } from "lodash";

import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";

import UpdateSoftwareItem from "./UpdateSoftwareItem";

describe("Self-service - UpdateSoftwareItem component", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  it("renders software name, version, and icon", () => {
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware(),
      ui_status: "update_available",
    };
    const { container } = render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={noop}
      />
    );
    // Name, version, and image alt attribute/assertions:
    expect(screen.getAllByText("mock software 1.app").length).toBeGreaterThan(
      0
    ); // Tooltip has text showing twice so can't use getByText
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    // Software icon
    expect(
      container.querySelector("svg.software-icon.software-icon__large")
    ).toBeInTheDocument();
  });

  it("shows 'Update' button and does not show status when installed", () => {
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({ status: "installed" }),
      ui_status: "update_available",
    };
    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={noop}
      />
    );
    expect(screen.getByRole("button", { name: "Update" })).toBeEnabled();
    // Should not show error or spinner status parts
    expect(
      screen.queryByTestId("update-software-item__status--test")
    ).not.toBeInTheDocument();
  });

  it("shows updating spinner and disables update when ui_status is 'updating'", () => {
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({ status: "pending_install" }),
      ui_status: "updating",
    };
    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={noop}
      />
    );

    const loadingSpinner = screen.getByTestId("spinner");
    expect(loadingSpinner).toBeVisible();
    expect(screen.getByText(/Updating.../)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Update" })
    ).not.toBeInTheDocument();
    // Spinner should exist (by role or other query, depending on how your Spinner renders)
  });

  it("shows install failed status and a 'Failed' button", async () => {
    const handleShowDetails = jest.fn();

    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({
        status: "failed_install",
      }),
      ui_status: "failed_install_update_available",
    };

    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={handleShowDetails}
      />
    );
    // Button should exist ("Failed" text is rendered as button)
    const failedButton = screen.getByRole("button", { name: "Failed" });
    expect(failedButton).toBeInTheDocument();

    expect(screen.getByText(/Software failed to install/)).toBeInTheDocument();
  });

  it("renders 'Updated' state when ui_status is 'recently_updated'", () => {
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({ status: "installed" }),
      ui_status: "recently_updated",
    };

    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={noop}
      />
    );

    // Updated icon/text instead of Update button
    expect(screen.getByText("Updated")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Update" })
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("success-icon")).toBeInTheDocument();
  });

  it("calls onClickUpdateAction when Update button is clicked", () => {
    const handleUpdate = jest.fn();
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({ status: "installed" }),
      ui_status: "update_available",
    };

    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={handleUpdate}
        onShowInstallerDetails={noop}
      />
    );

    const updateButton = screen.getByRole("button", { name: "Update" });
    updateButton.click();

    expect(handleUpdate).toHaveBeenCalledWith(software.id);
  });

  it("calls onShowInstallerDetails when Failed button is clicked", () => {
    const handleShowDetails = jest.fn();
    const software: IDeviceSoftwareWithUiStatus = {
      ...createMockDeviceSoftware({ status: "failed_install" }),
      ui_status: "failed_install_update_available",
    };

    render(
      <UpdateSoftwareItem
        software={software}
        onClickUpdateAction={noop}
        onShowInstallerDetails={handleShowDetails}
      />
    );

    const failedButton = screen.getByRole("button", { name: "Failed" });
    failedButton.click();

    expect(handleShowDetails).toHaveBeenCalled();
  });
});
