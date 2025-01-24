import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";

import FleetAppDetailsModal from "./FleetAppDetailsModal";

describe("FleetAppDetailsModal", () => {
  const defaultProps = {
    name: "Test App",
    platform: "macOS",
    version: "1.0.0",
    url: "https://example.com/app",
  };

  it("renders modal with correct title", () => {
    const render = createCustomRenderer();

    render(<FleetAppDetailsModal {...defaultProps} />);

    const modalTitle = screen.getByText("Software details");
    expect(modalTitle).toBeInTheDocument();
  });

  it("displays correct app details", () => {
    const render = createCustomRenderer();

    render(<FleetAppDetailsModal {...defaultProps} />);

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Test App")).toBeInTheDocument();
    expect(screen.getByText("Platform")).toBeInTheDocument();
    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("URL")).toBeInTheDocument();
    expect(screen.getByText("https://example.com/app")).toBeInTheDocument();
  });

  it("does not render URL field when url prop is not provided", () => {
    const render = createCustomRenderer();
    const propsWithoutUrl = { ...defaultProps, url: undefined };

    render(<FleetAppDetailsModal {...propsWithoutUrl} />);

    expect(screen.queryByText("URL")).not.toBeInTheDocument();
  });
});
