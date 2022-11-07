import React from "react";

import { Router } from "react-router";
// import { createMemoryHistory } from "history";
import { render, screen } from "@testing-library/react";
// import { renderWithSetup } from "test/testingUtils";

import BackLink from "./BackLink";

describe("BackLink - component", () => {
  it("renders text and icon", () => {
    render(<BackLink text="Back to software" />);

    const text = screen.getByText("Back to software");
    const icon = screen.getByTestId("Icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
  it("renders link on click", async () => {
    // const { user } = renderWithSetup(<BackLink text="Back to software" />);
    // one approach
    //     renderWithRouter(
    //   ui,
    //   {
    //     route = "/",
    //     history = createMemoryHistory({ initialEntries: [route] }),
    //   } = {}
    // ) {
    //   return {
    //     ...render(<Router history={history}>{ui}</Router>),
    //     history,
    //   };
    // another approach
    // await user.click(screen.getByText("Back to software"));
    // TODO: how to test a back link
    // expect(window.location.pathname).toBe(
    //   "https://github.com/fleetdm/fleet/issues/new/choose"
    // );
  });
});

// import React from "react";
// import { Router } from "react-router-dom";
// import { render } from "@testing-library/react";
// import { createMemoryHistory } from "history";

// function renderWithRouter(
//   ui,
//   {
//     route = "/",
//     history = createMemoryHistory({ initialEntries: [route] }),
//   } = {}
// ) {
//   return {
//     ...render(<Router history={history}>{ui}</Router>),
//     history,
//   };
// }
