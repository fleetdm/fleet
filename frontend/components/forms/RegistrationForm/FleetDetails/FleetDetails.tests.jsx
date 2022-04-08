import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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

  it("validates the presence of the fleet web address field", () => {
    render(<FleetDetails handleSubmit={handleSubmitSpy} currentPage />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Fleet web address must be completed")
    ).toBeInTheDocument();
  });

  it("validates the fleet web address field starts with https://", () => {
    render(<FleetDetails handleSubmit={handleSubmitSpy} currentPage />);
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Fleet web address" }),
      "http://gnar.Fleet.co"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Fleet web address must start with https://")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", () => {
    render(<FleetDetails handleSubmit={handleSubmitSpy} currentPage />);
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Fleet web address" }),
      "https://gnar.Fleet.co"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      server_url: "https://gnar.Fleet.co",
    });
  });
});
