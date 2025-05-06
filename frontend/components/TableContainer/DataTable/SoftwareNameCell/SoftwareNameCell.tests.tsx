import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import SoftwareNameCell from "./SoftwareNameCell";

// TODO: figure out how to mock the router properly.
const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

describe("SoftwareNameCell", () => {
  const defaultProps = {
    name: "Fleet Desktop",
    source: "fleet",
  };

  it("renders a non-clickable cell when no router or path is provided", () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} />);
    expect(screen.getAllByText(/Fleet Desktop/i).length).toBeGreaterThan(0);
    // Should not render as a link
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("renders with a suffix icon and tooltip when hasPackage is true", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        router={mockRouter}
        path="/software/1"
        hasPackage
        isSelfService={false}
        installType="manual"
      />
    );
    expect(
      screen.getByText("Software can be installed on Host details page.")
    ).toBeInTheDocument();
  });

  it("renders the correct count for tooltip for automatic installType", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        router={mockRouter}
        path="/software/1"
        hasPackage
        installType="automatic"
        automaticInstallPoliciesCount={2}
      />
    );
    expect(screen.getByText("2 policies trigger install.")).toBeInTheDocument();
  });

  it("renders the correct tooltip for automaticSelfService", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        router={mockRouter}
        path="/software/1"
        hasPackage
        isSelfService
        installType="automatic"
        automaticInstallPoliciesCount={1}
      />
    );
    expect(screen.getByText(/A policy triggers install./)).toBeInTheDocument();
    expect(screen.getByText(/End users can reinstall/)).toBeInTheDocument();
  });

  it("renders with SoftwareIcon and truncated tooltip on myDevicePage", () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} myDevicePage />);
    expect(screen.getAllByText(/Fleet Desktop/i).length).toBeGreaterThan(0);
    // Should not render as a link
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("renders the self-service tooltip when isSelfService is true and installType is not automatic", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        router={mockRouter}
        path="/software/1"
        hasPackage
        isSelfService
        installType="manual"
      />
    );

    // Check for the SELF_SERVICE_TOOLTIP content
    expect(screen.getByText(/End users can install/)).toBeInTheDocument();
  });
});
