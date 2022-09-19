import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";

import paths from "router/paths";

import SummaryTile from "./SummaryTile";

import TestIcon from "../../../../../assets/images/icon-windows-black-24x24@2x.png";

const INITIAL_OPACITY = 0;

const LOADING_OPACITY = 0.4;

describe("SummaryTile - component", () => {
  it("summary tile is hidden when showUI is false on first load", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI={false} // being tested
        title={"Windows hosts"}
        icon={TestIcon}
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).toHaveStyle(`opacity: ${INITIAL_OPACITY}`);
  });

  it("renders loading state", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={true} // tested
        showUI={true}
        title={"Windows hosts"}
        icon={TestIcon}
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).toHaveStyle(`opacity: ${LOADING_OPACITY}`);
  });

  it("renders title, count, and image based on the information and data passed in", () => {
    render(
      <SummaryTile
        count={200} // tested
        isLoading={false}
        showUI={true}
        title={"Windows hosts"} // tested
        icon={TestIcon} // tested
        tooltip={"Hosts on any Windows device"}
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const title = screen.getByText("Windows hosts");

    const count = screen.getByText("200");

    const icon = screen.getByRole("img");

    expect(title).toBeInTheDocument();

    expect(count).toBeInTheDocument();

    expect(icon).toHaveAttribute("src", "test-file-stub");
  });

  it("renders tooltip on title hover", async () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI={true}
        title={"Windows hosts"}
        icon={TestIcon}
        tooltip={"Hosts on any Windows device"} // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    fireEvent.mouseOver(screen.getByText("Windows hosts"));

    expect(
      await screen.findByText("Hosts on any Windows device")
    ).toBeInTheDocument();
  });

  it("renders manage host page on click", async () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI={true}
        title={"Windows hosts"}
        icon={TestIcon}
        tooltip={"Hosts on any Windows device"} // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    fireEvent.click(screen.getByText("Windows hosts"));

    expect(window.location.pathname).toBe("/hosts/manage/labels/10");
  });
});
