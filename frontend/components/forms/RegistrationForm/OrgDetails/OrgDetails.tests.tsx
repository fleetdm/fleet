import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
import OrgDetails from "components/forms/RegistrationForm/OrgDetails";

describe("OrgDetails - form", () => {
  const handleSubmitSpy = jest.fn();

  beforeEach(() => {
    handleSubmitSpy.mockReset();
  });

  it("renders", () => {
    render(<OrgDetails handleSubmit={handleSubmitSpy} currentPage />);

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

  it("submits with the org name and a null logo when no file is selected", async () => {
    const { user } = renderWithSetup(
      <OrgDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Organization name" }),
      "The Gnar Co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).toHaveBeenCalledWith({
      org_name: "The Gnar Co",
      org_logo_file: null,
    });
  });
});
