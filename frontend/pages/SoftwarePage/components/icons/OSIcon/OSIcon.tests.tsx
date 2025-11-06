// OSIcon.test.tsx
import React from "react";
import { render, screen } from "@testing-library/react";

import { SOFTWARE_ICON_SIZES } from "styles/var/icon_sizes";
import OSIcon from "./OSIcon";
import { getMatchedOsIcon } from "..";

// Mock getMatchedOsIcon to return a fake icon component
jest.mock("..", () => ({
  getMatchedOsIcon: jest.fn(),
}));

// Create a simple mock SVG component for matched icons
const MockSvgIcon = ({ width, height, className }: any) => (
  <svg
    data-testid="mock-svg"
    width={width}
    height={height}
    className={className}
  />
);

describe("OSIcon", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (getMatchedOsIcon as jest.Mock).mockReturnValue(MockSvgIcon);
  });

  it("renders an <img> with the correct src when url is provided", () => {
    render(<OSIcon url="https://example.com/icon.png" size="large" />);
    const img = screen.getByRole("presentation"); // instead of "img" when no alt text
    expect(img).toHaveAttribute("src", "https://example.com/icon.png");
    expect(img).toHaveClass("os-icon__os-img-large");
  });

  it("renders matched OS icon SVG when url is not provided", () => {
    render(<OSIcon name="ubuntu" size="small" />);

    expect(getMatchedOsIcon).toHaveBeenCalledWith({ name: "ubuntu" });

    const svg = screen.getByTestId("mock-svg");
    expect(svg).toBeInTheDocument();
    expect(svg).toHaveAttribute("width", SOFTWARE_ICON_SIZES.small.toString());
    expect(svg).toHaveAttribute("height", SOFTWARE_ICON_SIZES.small.toString());
    expect(svg).toHaveClass("os-icon os-icon__small");
  });

  it("uses default size 'small' when size is not specified", () => {
    render(<OSIcon name="windows" />);
    const svg = screen.getByTestId("mock-svg");
    expect(svg).toHaveAttribute("width", SOFTWARE_ICON_SIZES.small.toString());
  });

  it("passes empty string as name when not provided", () => {
    render(<OSIcon />);
    expect(getMatchedOsIcon).toHaveBeenCalledWith({ name: "" });
  });
});
