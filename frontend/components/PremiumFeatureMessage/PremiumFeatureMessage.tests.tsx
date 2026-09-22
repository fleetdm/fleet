import { render, screen } from "@testing-library/react";
import React from "react";

import PremiumFeatureMessage from "./PremiumFeatureMessage";

describe("PremiumFeatureMessage - component", () => {
  it("renders the page message, learn more link, and ghost table (Option B)", () => {
    const { container } = render(<PremiumFeatureMessage />);

    expect(screen.getByText("Included in Fleet Premium")).toBeInTheDocument();

    const link = screen.getByText("Learn more").closest("a");
    expect(link).toHaveAttribute("href", "https://fleetdm.com/upgrade");
    expect(link).toHaveAttribute("target", "_blank");

    const rootElement = container.firstChild as HTMLElement;
    expect(rootElement).toHaveClass("premium-feature-message-container");
    expect(rootElement).not.toHaveClass(
      "premium-feature-message-container--compact"
    );
    expect(
      container.querySelector(".premium-feature-message-container__ghost-table")
    ).toBeInTheDocument();
  });

  it("renders the compact card layout with full sentence (Option A)", () => {
    const { container } = render(<PremiumFeatureMessage variant="compact" />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();

    const rootElement = container.firstChild as HTMLElement;
    expect(rootElement).toHaveClass(
      "premium-feature-message-container",
      "premium-feature-message-container--compact"
    );
    expect(
      container.querySelector(".premium-feature-message")
    ).toBeInTheDocument();
    expect(
      container.querySelector(".premium-feature-message-container__ghost-table")
    ).not.toBeInTheDocument();
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
