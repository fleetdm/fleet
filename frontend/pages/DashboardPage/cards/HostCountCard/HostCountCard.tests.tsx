import React from "react";

import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import paths from "router/paths";
import HostCountCard from "./HostCountCard";

describe("HostCountCard - component", () => {
  it("renders title, count, and image based on the information and data passed in", () => {
    render(
      <HostCountCard
        count={200} // tested
        title="Windows hosts" // tested
        iconName="windows" // tested
        tooltip="Hosts on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const title = screen.getByText("Windows hosts");
    const count = screen.getByText("200");
    const icon = screen.queryByTestId("windows-icon");

    expect(title).toBeInTheDocument();
    expect(count).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("does not render icon if not provided", () => {
    render(
      <HostCountCard
        count={200}
        title="Windows hosts"
        iconName="windows"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const icon = screen.queryByRole("svg");

    expect(icon).toBeNull();
  });

  it("renders tooltip on title hover", async () => {
    const { user } = renderWithSetup(
      <HostCountCard
        count={200}
        title="Windows hosts"
        iconName="windows"
        tooltip="Hosts on any Windows device" // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    await user.hover(screen.getByText("Windows hosts"));

    await waitFor(() => {
      const tooltip = screen.getByText("Hosts on any Windows device");
      expect(tooltip).toBeInTheDocument();
    });
  });

  // Note: Cannot test path of react-router <Link/> without <Router/>
});
