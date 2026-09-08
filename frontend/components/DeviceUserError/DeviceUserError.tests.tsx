import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

  it("renders SSO failure copy, and its retry, ahead of the authentication error", async () => {
    const onRetry = jest.fn();
    render(
      <DeviceUserError
        isAuthenticationError
        ssoError="sign_in_failed"
        onRetry={onRetry}
      />
    );
    expect(screen.getByText("Couldn't sign in.")).toBeInTheDocument();
    expect(
      screen.queryByText("This URL is invalid or expired.")
    ).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: "Sign in again" })
    );
    expect(onRetry).toHaveBeenCalled();
  });

  it("omits the retry where the end user has no way to retry", () => {
    render(<DeviceUserError ssoError="session_expired" />);
    expect(
      screen.getByText("Your sign-in session expired.")
    ).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
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
