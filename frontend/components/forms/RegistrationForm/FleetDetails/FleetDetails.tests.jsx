import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import FleetDetails from "components/forms/RegistrationForm/FleetDetails";

import INVALID_SERVER_URL_MESSAGE from "utilities/error_messages";

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

  it("validates the Fleet server URL field starts with 'https://' or 'http://'", async () => {
    const { user } = renderWithSetup(
      <FleetDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    const inputField = screen.getByRole("textbox", {
      name: "Fleet web address",
    });
    const nextButton = screen.getByRole("button", { name: "Next" });

    await user.type(inputField, "gnar.Fleet.co");
    await user.click(nextButton);

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText(INVALID_SERVER_URL_MESSAGE)).toBeInTheDocument();

    await user.type(inputField, "localhost:8080");
    await user.click(nextButton);

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText(INVALID_SERVER_URL_MESSAGE)).toBeInTheDocument();
  });

  it("submits the form with valid https link", async () => {
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
  it("submits the form with valid http link", async () => {
    const { user } = renderWithSetup(
      <FleetDetails handleSubmit={handleSubmitSpy} currentPage />
    );
    // when
    await user.type(
      screen.getByRole("textbox", { name: "Fleet web address" }),
      "http://localhost:8080"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      server_url: "http://localhost:8080",
    });
  });
});
