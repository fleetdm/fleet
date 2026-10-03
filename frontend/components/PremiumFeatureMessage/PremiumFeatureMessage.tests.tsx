import { render, screen } from "@testing-library/react";
import React from "react";

import PremiumFeatureMessage from "./PremiumFeatureMessage";

const expectUpgradeLink = () => {
  const link = screen.getByRole("link", { name: /learn more/i });
  expect(link).toHaveAttribute("href", "https://fleetdm.com/upgrade");
  expect(link).toHaveAttribute("target", "_blank");
};

describe("PremiumFeatureMessage - component", () => {
  it("renders the empty-state heading and upgrade link by default", () => {
    render(<PremiumFeatureMessage />);

    expect(
      screen.getByRole("heading", { name: "Included in Fleet Premium" })
    ).toBeInTheDocument();
    expect(
      screen.queryByText("This feature is included in Fleet Premium.")
    ).not.toBeInTheDocument();
    expectUpgradeLink();
  });

  it("renders the inline sentence and upgrade link in the compact variant", () => {
    render(<PremiumFeatureMessage variant="compact" />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading")).not.toBeInTheDocument();
    expectUpgradeLink();
  });
});
