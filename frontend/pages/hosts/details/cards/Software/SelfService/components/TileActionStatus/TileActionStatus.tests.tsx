import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { createMockHostSoftware } from "__mocks__/hostMock";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";

import TileActionStatus from "./TileActionStatus";

// Helper to create software status objects
const makeSoftware = (ui_status: string) =>
  ({
    ...createMockHostSoftware(),
    ui_status,
  } as IDeviceSoftwareWithUiStatus);

describe("TileActionStatus", () => {
  // Active/pending/running statuses
  it("shows spinner and installing label if installing", () => {
    render(
      <TileActionStatus
        software={makeSoftware("installing")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Installing.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and installing label if pending_install", () => {
    render(
      <TileActionStatus
        software={makeSoftware("pending_install")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Installing.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and updating label if updating", () => {
    render(
      <TileActionStatus
        software={makeSoftware("updating")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Updating.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and updating label if pending_update", () => {
    render(
      <TileActionStatus
        software={makeSoftware("pending_update")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Updating.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and running label if running_script", () => {
    render(
      <TileActionStatus
        software={makeSoftware("running_script")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Running.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and running label if pending_script", () => {
    render(
      <TileActionStatus
        software={makeSoftware("pending_script")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Running.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and uninstalling label if uninstalling", () => {
    render(
      <TileActionStatus
        software={makeSoftware("uninstalling")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Uninstalling.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows spinner and uninstalling label if pending_uninstall", () => {
    render(
      <TileActionStatus
        software={makeSoftware("pending_uninstall")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByText(/Uninstalling.../i)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  // Button/status tests
  it("shows Install button for uninstalled status", () => {
    const onClick = jest.fn();
    render(
      <TileActionStatus
        software={makeSoftware("uninstalled")}
        onActionClick={onClick}
      />
    );
    const button = screen.getByRole("button", { name: /Install/i });
    expect(button).toBeInTheDocument();
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalled();
  });

  it("shows Install button for recently_uninstalled status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("recently_uninstalled")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Install/i })
    ).toBeInTheDocument();
  });

  it("shows Update button for update_available status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("update_available")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Update/i })).toBeInTheDocument();
  });

  it("shows Update button for failed_uninstall_update_available status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("failed_uninstall_update_available")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Update/i })).toBeInTheDocument();
  });

  it("shows Reinstall button for installed status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("installed")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Reinstall/i })
    ).toBeInTheDocument();
  });

  it("shows Reinstall button for recently_installed status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("recently_installed")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Reinstall/i })
    ).toBeInTheDocument();
  });

  it("shows Reinstall button for recently_updated status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("recently_updated")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Reinstall/i })
    ).toBeInTheDocument();
  });

  it("shows Reinstall button for failed_uninstall status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("failed_uninstall")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Reinstall/i })
    ).toBeInTheDocument();
  });

  it("shows Run button for never_ran_script status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("never_ran_script")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Run/i })).toBeInTheDocument();
  });

  it("shows Rerun button for ran_script status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("ran_script")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Rerun/i })).toBeInTheDocument();
  });

  it("shows Retry button and error display for failed_install", () => {
    render(
      <TileActionStatus
        software={makeSoftware("failed_install")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Retry/i })).toBeInTheDocument();
    expect(screen.getByText(/Failed/i)).toBeInTheDocument();
    expect(screen.getByTestId("error-icon")).toBeInTheDocument();
  });

  it("shows Retry and error for failed_install_update_available", () => {
    render(
      <TileActionStatus
        software={makeSoftware("failed_install_update_available")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Retry/i })).toBeInTheDocument();
    expect(screen.getByText(/Failed/i)).toBeInTheDocument();
    expect(screen.getByTestId("error-icon")).toBeInTheDocument();
  });

  it("shows Retry button for failed_script status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("failed_script")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Retry/i })).toBeInTheDocument();
  });

  // Unknown/fallback status
  it("shows Install button for unknown_status (default)", () => {
    render(
      <TileActionStatus
        software={makeSoftware("unknown_status")}
        onActionClick={jest.fn()}
      />
    );
    expect(
      screen.getByRole("button", { name: /Install/i })
    ).toBeInTheDocument();
  });
});
