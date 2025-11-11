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

  it("shows Update button for update_available status", () => {
    render(
      <TileActionStatus
        software={makeSoftware("update_available")}
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

  it("renders nothing if tile label is null", () => {
    render(
      <TileActionStatus
        software={makeSoftware("unknown_status")}
        onActionClick={jest.fn()}
      />
    );
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByText(/Failed/i)).not.toBeInTheDocument();
  });
});
