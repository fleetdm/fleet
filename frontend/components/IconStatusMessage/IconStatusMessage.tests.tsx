import React from "react";
import { render, screen } from "@testing-library/react";
import IconStatusMessage from "./IconStatusMessage";

describe("IconStatusMessage - component", () => {
  it("renders with icon and message", () => {
    const { container } = render(
      <IconStatusMessage
        message="Test message"
        iconName="success"
        iconColor="core-fleet-green"
      />
    );
    expect(
      container.querySelector(".icon-status-message__icon")
    ).toBeInTheDocument();
    expect(screen.getByTestId("success-icon")).toBeInTheDocument();
    expect(
      container.querySelector(".icon-status-message__content")
    ).toHaveTextContent("Test message");
  });

  it("does not render icon when iconName is not provided", () => {
    const { container } = render(<IconStatusMessage message="No icon" />);
    expect(container.querySelector(".icon-status-message__icon")).toBeNull();
    expect(
      container.querySelector(".icon-status-message__content")
    ).toHaveTextContent("No icon");
  });

  it("applies custom className and testId when provided", () => {
    const { container } = render(
      <IconStatusMessage
        message="Custom class"
        className="extra-style"
        testId="status-msg"
      />
    );
    expect(container.firstChild).toHaveClass("icon-status-message");
    expect(container.firstChild).toHaveClass("extra-style");
    expect(screen.getByTestId("status-msg")).toBeInTheDocument();
  });
});
