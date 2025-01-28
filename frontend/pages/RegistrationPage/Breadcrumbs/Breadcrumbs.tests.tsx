import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { noop } from "lodash";

import Breadcrumbs from "pages/RegistrationPage/Breadcrumbs";

describe("Breadcrumbs - component", () => {
  it("renders 3 Button components", () => {
    render(<Breadcrumbs onSetPage={noop} pageProgress={1} currentPage={1} />);
    expect(screen.getAllByRole("button").length).toEqual(3);
  });

  it("renders page 1 Button as active when the current page prop is 1", () => {
    const { container } = render(
      <Breadcrumbs onSetPage={noop} pageProgress={1} currentPage={1} />
    );
    const page1Btn = container.querySelector(
      "button.registration-breadcrumbs__page--1"
    );
    const page2Btn = container.querySelector(
      "button.registration-breadcrumbs__page--2"
    );
    const page3Btn = container.querySelector(
      "button.registration-breadcrumbs__page--3"
    );

    expect(page1Btn?.className).toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page2Btn?.className).not.toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page3Btn?.className).not.toContain(
      "registration-breadcrumbs__page--active"
    );
  });
});
