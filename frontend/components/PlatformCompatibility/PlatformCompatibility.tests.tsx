import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import PlatformCompatibility from "./PlatformCompatibility";

describe("Platform compatibility", () => {
  it("renders compatible platforms", () => {
    render(
      <PlatformCompatibility
        compatiblePlatforms={["macOS", "Windows"]}
        error={null}
      />
    );
    const macCompatibility = screen.getByText("macOS").firstElementChild;
    const windowsCompatibility = screen.getByText("Windows").firstElementChild;
    const linuxCompatibility = screen.getByText("Linux").firstElementChild;

    expect(macCompatibility).toHaveAttribute("alt", "compatible");
    expect(windowsCompatibility).toHaveAttribute("alt", "compatible");
    expect(linuxCompatibility).toHaveAttribute("alt", "incompatible");
  });
  // it("renders error state", () => {
  //   render(<PlatformCompatibility whatToRetrieve="software" />);

  //   const text = screen.getByText("Updated never");

  //   expect(text).toBeInTheDocument();
  // });

  // it("renders tooltip on hover", async () => {
  //   const { user } = renderWithSetup(
  //     <LastUpdatedText whatToRetrieve="software" />
  //   );

  //   await user.hover(screen.getByText("Updated never"));

  //   expect(screen.getByText(/to retrieve software/i)).toBeInTheDocument();
  // });
});
