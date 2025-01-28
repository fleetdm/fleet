import React from "react";
import { render, screen } from "@testing-library/react";

import PlatformCompatibility from "./PlatformCompatibility";

describe("Platform compatibility", () => {
  it("renders compatible platforms", () => {
    render(
      <PlatformCompatibility
        compatiblePlatforms={["darwin", "windows"]}
        error={null}
      />
    );
    const macCompatibility = screen.getByText("macOS").firstElementChild;
    const windowsCompatibility = screen.getByText("Windows").firstElementChild;
    const linuxCompatibility = screen.getByText("Linux").firstElementChild;

    expect(macCompatibility).toHaveAttribute(
      "class",
      "icon compatible-platform"
    );
    expect(windowsCompatibility).toHaveAttribute(
      "class",
      "icon compatible-platform"
    );
    expect(linuxCompatibility).toHaveAttribute(
      "class",
      "icon incompatible-platform"
    );
  });
  it("renders empty state", () => {
    render(<PlatformCompatibility compatiblePlatforms={[]} error={null} />);

    const text = screen.getByText(/No platforms/i);

    expect(text).toBeInTheDocument();
  });
  it("renders error state", () => {
    render(
      <PlatformCompatibility
        compatiblePlatforms={["macos"]}
        error={{ name: "Error", message: "The resource was not found." }}
      />
    );

    const text = screen.getByText(/possible syntax error/i);

    expect(text).toBeInTheDocument();
  });
});
