import React from "react";
import { render, screen } from "@testing-library/react";
import CustomLink from "./CustomLink";

describe("CustomLink - component", () => {
  it("renders text, link in same tab, and no icon", () => {
    render(
      <CustomLink
        url="https://github.com/fleetdm/fleet/issues/new/choose"
        text="file an issue"
      />
    );

    const text = screen.getByText("file an issue");
    const icon = screen.queryByTestId("Icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeNull();
    expect(text.closest("a")).toHaveAttribute(
      "href",
      "https://github.com/fleetdm/fleet/issues/new/choose"
    );
    expect(text.closest("a")).not.toHaveAttribute("target", "_blank");
  });

  it("renders icon and link in new tab if newTab is set", () => {
    render(
      <CustomLink
        url="https://github.com/fleetdm/fleet/issues/new/choose"
        text="file an issue"
        newTab
      />
    );

    const icon = screen.getByTestId("external-link-icon");

    expect(icon).toBeInTheDocument();
    expect(icon.closest("a")).toHaveAttribute("target", "_blank");
  });
});
