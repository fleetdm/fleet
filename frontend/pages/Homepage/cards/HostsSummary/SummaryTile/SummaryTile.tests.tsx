import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";
import paths from "router/paths";
import SummaryTile from "./SummaryTile";

const LOADING_OPACITY = 0.4;

describe("SummaryTile - component", () => {
  it("summary tile is hidden when showUI is false", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI={false} // tested
        title={"Windows hosts"}
        iconName={"windows-blue"}
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).not.toBeVisible();
  });

  it("renders loading state", () => {
    render(
      <SummaryTile
        count={200}
        isLoading // tested
        showUI
        title={"Windows hosts"}
        iconName={"windows-blue"}
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).toHaveStyle(`opacity: ${LOADING_OPACITY}`);
    expect(tile).toBeVisible();
  });

  it("renders title, count, and image based on the information and data passed in", () => {
    render(
      <SummaryTile
        count={200} // tested
        isLoading={false}
        showUI
        title={"Windows hosts"} // tested
        iconName={"windows-blue"} // tested
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const title = screen.getByText("Windows hosts");
    const count = screen.getByText("200");
    // TOOD: Fix icon assertion
    // const icon = screen.getByRole("svg");

    expect(title).toBeInTheDocument();
    expect(count).toBeInTheDocument();
    // expect(icon).toBeInTheDocument();
  });

  it("does not render icon if not provided", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI
        title={"Windows hosts"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const icon = screen.queryByRole("img");

    expect(icon).toBeNull();
  });

  it("renders tooltip on title hover", async () => {
    const { user } = renderWithSetup(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI
        title={"Windows hosts"}
        iconName={"windows-blue"}
        tooltip={"Hosts on any Windows device"} // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    await user.hover(screen.getByText("Windows hosts"));

    expect(screen.getByText("Hosts on any Windows device")).toBeInTheDocument();
  });

  it("renders manage host page on click", async () => {
    const { user } = renderWithSetup(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI
        title={"Windows hosts"}
        iconName={"windows-blue"}
        tooltip={"Hosts on any Windows device"} // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    await user.click(screen.getByText("Windows hosts"));

    expect(window.location.pathname).toBe("/hosts/manage/labels/10");
  });
});
