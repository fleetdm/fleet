import { render, screen } from "@testing-library/react";
import React from "react";

import PremiumFeatureMessage from "./PremiumFeatureMessage";

describe("PremiumFeatureMessage - component", () => {
  it("renders the message text and learn more link", () => {
    render(<PremiumFeatureMessage />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();

    const link = screen.getByText("Learn more").closest("a");
    expect(link).toHaveAttribute("href", "https://fleetdm.com/upgrade");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders with default variant container classes", () => {
    const { container } = render(<PremiumFeatureMessage />);

    const rootElement = container.firstChild as HTMLElement;
    expect(rootElement).toHaveClass("premium-feature-message-container");
    expect(rootElement).not.toHaveClass(
      "premium-feature-message-container--compact"
    );
  });

  it("renders with compact variant container class", () => {
    const { container } = render(<PremiumFeatureMessage variant="compact" />);

    const rootElement = container.firstChild as HTMLElement;
    expect(rootElement).toHaveClass(
      "premium-feature-message-container",
      "premium-feature-message-container--compact"
    );
  });

  it("applies custom className", () => {
    const { container } = render(
      <PremiumFeatureMessage className="custom-class" />
    );

    const rootElement = container.firstChild as HTMLElement;
    expect(rootElement).toHaveClass(
      "premium-feature-message-container",
      "custom-class"
    );
  });
});
