import React from "react";
import { render, screen } from "@testing-library/react";

import RegistrationForm from "components/forms/RegistrationForm";

describe("RegistrationForm - component", () => {
  it("renders AdminDetails and header on the first page", () => {
    const { container } = render(<RegistrationForm page={1} />);

    expect(
      container.querySelectorAll(".user-registration__container--admin").length
    ).toEqual(1);
    expect(screen.getByText("Set up user")).toBeInTheDocument();
  });

  it("renders OrgDetails on the second page", () => {
    const { container } = render(<RegistrationForm page={2} />);

    expect(
      container.querySelectorAll(".user-registration__container--org").length
    ).toEqual(1);
    expect(screen.getByText("Organization details")).toBeInTheDocument();
  });

  it("renders FleetDetails on the third page", () => {
    const { container } = render(<RegistrationForm page={3} />);

    expect(
      container.querySelectorAll(".user-registration__container--fleet").length
    ).toEqual(1);
    expect(screen.getByText("Set Fleet URL")).toBeInTheDocument();
  });

  it("renders ConfirmationPage on the fourth page", () => {
    const { container } = render(<RegistrationForm page={4} />);

    expect(
      container.querySelectorAll(".user-registration__container--confirmation")
        .length
    ).toEqual(1);
    expect(screen.getByText("Confirm configuration")).toBeInTheDocument();
  });
});
