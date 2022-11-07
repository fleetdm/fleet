import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import ViewAllHostsLink from "./ViewAllHostsLink";

describe("ViewAllHostsLink - component", () => {
  it("renders View all hosts text and icon", () => {
    render(<ViewAllHostsLink />);

    const title = screen.getByText("View all hosts");
    const icon = screen.queryByTitle("Icon");

    expect(title).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("renders link on click", async () => {
    // const { user } = renderWithSetup(
    //   <ViewAllHostsLink queryParams={{ status: "online" }} />
    // );

    // await user.click(screen.getByText("View all hosts"));

    // // TODO: how to test a link
    // expect(window.location.pathname).toBe("/hosts/manage/&status=online");

    render(<ViewAllHostsLink queryParams={{ status: "online" }} />);

    const links: HTMLAnchorElement[] = screen.getAllByTitle("host-link");

    console.log("links", links);
    expect(links[0].textContent).toEqual("View all hosts");
    expect(links[0].href).toContain("/hosts/manage/&status=online");
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} condensed />);
    const title = screen.queryByText("View all hosts");

    expect(title).toBeNull();
  });
});
