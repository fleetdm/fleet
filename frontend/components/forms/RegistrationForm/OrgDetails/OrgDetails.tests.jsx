import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
import OrgDetails from "components/forms/RegistrationForm/OrgDetails";

describe("OrgDetails - form", () => {
  const handleSubmitSpy = jest.fn();
  it("renders", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} />);

    expect(
      screen.getByRole("textbox", { name: "Organization name" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates presence of org_name field", async () => {
    const { user } = renderWithSetup(
      <OrgDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Organization name must be present")
    ).toBeInTheDocument();
  });

  it("validates the logo url field starts with https://", async () => {
    const { user } = renderWithSetup(
      <OrgDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Organization logo URL (optional)" }),
      "http://www.thegnar.co/logo.png"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Organization logo URL must start with https://")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", async () => {
    const { user } = renderWithSetup(
      <OrgDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Organization logo URL (optional)" }),
      "https://www.thegnar.co/logo.png"
    );
    await user.type(
      screen.getByRole("textbox", { name: "Organization name" }),
      "The Gnar Co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).toHaveBeenCalledWith({
      org_logo_url: "https://www.thegnar.co/logo.png",
      org_name: "The Gnar Co",
    });
  });
});
