import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";
import paths from "router/paths";
import HostCountCard from "./HostCountCard";

const LOADING_OPACITY = 0.4;

describe("HostCountCard - component", () => {
  it("host count card is hidden when showUI is false", () => {
    render(
      <HostCountCard
        count={200}
        isLoading={false}
        showUI={false} // tested
        title="Windows hosts"
        iconName="windows"
        tooltip="Hosts on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const card = screen.getByTestId("card");

    expect(card).not.toBeVisible();
  });

  it("renders loading state", () => {
    render(
      <HostCountCard
        count={200}
        isLoading // tested
        showUI
        title="Windows hosts"
        iconName="windows"
        tooltip="Hosts on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const card = screen.getByTestId("card");

    expect(card).toHaveStyle(`opacity: ${LOADING_OPACITY}`);
    expect(card).toBeVisible();
  });

  it("renders title, count, and image based on the information and data passed in", () => {
    render(
      <HostCountCard
        count={200} // tested
        isLoading={false}
        showUI
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
        isLoading={false}
        showUI
        title="Windows hosts"
        iconName="windows"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const icon = screen.queryByRole("svg");

    expect(icon).toBeNull();
  });

  it("renders tooltip on title hover", async () => {
    render(
      <HostCountCard
        count={200}
        isLoading={false}
        showUI
        title="Windows hosts"
        iconName="windows"
        tooltip="Hosts on any Windows device" // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    await fireEvent.mouseEnter(screen.getByText("Windows hosts"));

    expect(screen.getByText("Hosts on any Windows device")).toBeInTheDocument();
  });

  // Note: Cannot test path of react-router <Link/> without <Router/>
});
