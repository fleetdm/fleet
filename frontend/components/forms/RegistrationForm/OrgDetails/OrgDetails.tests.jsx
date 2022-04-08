import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import OrgDetails from "components/forms/RegistrationForm/OrgDetails";
import userEvent from "@testing-library/user-event";

describe("OrgDetails - form", () => {
  const handleSubmitSpy = jest.fn();
  it("renders", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} />);

    expect(
      screen.getByRole("textbox", { name: "Organization name" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates presence of org_name field", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} currentPage />);
    // when
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Organization name must be present")
    ).toBeInTheDocument();
  });

  it("validates the logo url field starts with https://", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} currentPage />);
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Organization logo URL (optional)" }),
      "http://www.thegnar.co/logo.png"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Organization logo URL must start with https://")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} currentPage />);
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Organization logo URL (optional)" }),
      "https://www.thegnar.co/logo.png"
    );
    userEvent.type(
      screen.getByRole("textbox", { name: "Organization name" }),
      "The Gnar Co"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      org_logo_url: "https://www.thegnar.co/logo.png",
      org_name: "The Gnar Co",
    });
  });
});
