import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import Breadcrumbs from "pages/RegistrationPage/Breadcrumbs";

describe("Breadcrumbs - component", () => {
  it("renders 3 Button components", () => {
    render(<Breadcrumbs page={1} />);
    expect(screen.getAllByRole("button").length).toEqual(3);
  });

  it("renders page 1 Button as active when the page prop is 1", () => {
    const { container } = render(<Breadcrumbs page={1} />);
    const page1Btn = container.querySelector(
      "button.registration-breadcrumbs__page--1"
    );
    const page2Btn = container.querySelector(
      "button.registration-breadcrumbs__page--2"
    );
    const page3Btn = container.querySelector(
      "button.registration-breadcrumbs__page--3"
    );

    expect(page1Btn.className).toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page2Btn.className).not.toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page3Btn.className).not.toContain(
      "registration-breadcrumbs__page--active"
    );
  });

  it("calls the onClick prop with the page number when clicked", () => {
    const onClickSpy = jest.fn();
    const { container } = render(<Breadcrumbs page={1} onClick={onClickSpy} />);
    const page1Btn = container.querySelector(
      "button.registration-breadcrumbs__page--1"
    );
    const page2Btn = container.querySelector(
      "button.registration-breadcrumbs__page--2"
    );
    const page3Btn = container.querySelector(
      "button.registration-breadcrumbs__page--3"
    );

    fireEvent.click(page1Btn);

    expect(onClickSpy).toHaveBeenCalledWith(1);

    fireEvent.click(page2Btn);

    expect(onClickSpy).toHaveBeenCalledWith(2);

    fireEvent.click(page3Btn);

    expect(onClickSpy).toHaveBeenCalledWith(3);
  });
});
