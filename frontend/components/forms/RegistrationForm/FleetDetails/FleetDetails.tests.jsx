import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import FleetDetails from "components/forms/RegistrationForm/FleetDetails";

describe("FleetDetails - form", () => {
  const handleSubmitSpy = jest.fn();
  it("renders", () => {
    render(<FleetDetails handleSubmit={handleSubmitSpy} />);

    expect(
      screen.getByRole("textbox", { name: "Fleet web address" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates the presence of the fleet web address field", async () => {
    const { user } = renderWithSetup(
      <FleetDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Fleet web address must be completed")
    ).toBeInTheDocument();
  });

  it("validates the fleet web address field starts with https://", async () => {
    const { user } = renderWithSetup(
      <FleetDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Fleet web address" }),
      "http://gnar.Fleet.co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Fleet web address must start with https://")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", async () => {
    const { user } = renderWithSetup(
      <FleetDetails handleSubmit={handleSubmitSpy} currentPage />
    );
    // when
    await user.type(
      screen.getByRole("textbox", { name: "Fleet web address" }),
      "https://gnar.Fleet.co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      server_url: "https://gnar.Fleet.co",
    });
  });
});
