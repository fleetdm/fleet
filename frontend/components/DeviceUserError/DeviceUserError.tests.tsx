import React from "react";
import { render, screen } from "@testing-library/react";
import DeviceUserError from "./DeviceUserError";

describe("DeviceUserError", () => {
  it("renders default error message when no props given", () => {
    render(<DeviceUserError />);
    expect(screen.getByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(
      screen.getByText("Please contact your IT admin.")
    ).toBeInTheDocument();
  });

  it("applies mobile view class when isMobileView prop is true", () => {
    const { container } = render(<DeviceUserError isMobileView />);
    expect(container.firstChild).toHaveClass("device-user-error__mobile-view");
  });

  it("renders authentication error message on desktop device", () => {
    render(<DeviceUserError isAuthenticationError />);
    expect(
      screen.getByText("This URL is invalid or expired.")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/To access your device information, please click/i)
    ).toBeInTheDocument();
  });

  it("renders authentication error message on mobile device", () => {
    render(<DeviceUserError isAuthenticationError isMobileDevice />);
    expect(
      screen.getByText("Invalid or missing certificate")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Couldn't authenticate this device. Please contact your IT admin."
      )
    ).toBeInTheDocument();
  });
});
