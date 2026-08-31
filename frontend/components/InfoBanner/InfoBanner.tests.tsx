import React from "react";

import { render, screen, fireEvent } from "@testing-library/react";

import InfoBanner from "./InfoBanner";

describe("InfoBanner - component", () => {
  it("renders children content", () => {
    render(<InfoBanner>Info banner testing text</InfoBanner>);

    const title = screen.getByText("Info banner testing text");

    expect(title).toBeInTheDocument();
  });

  it("renders as page-level banner", () => {
    const { container } = render(<InfoBanner pageLevel />);
    expect(container.firstChild).toHaveClass("info-banner__page-banner");
  });

  it("renders CTA element", () => {
    const cta = <button>Click me</button>;
    render(<InfoBanner cta={cta} />);
    expect(screen.getByText("Click me")).toBeInTheDocument();
  });

  it("renders closable button and hides banner on click", () => {
    render(<InfoBanner closable>Test message</InfoBanner>);

    const closeButton = screen.getByRole("button");
    expect(closeButton).toBeInTheDocument();

    fireEvent.click(closeButton);
    expect(screen.queryByText("Test message")).not.toBeInTheDocument();
  });

  it("renders with icon class when icon prop is provided", () => {
    const { container } = render(<InfoBanner icon="info" />);
    expect(container.firstChild).toHaveClass("info-banner__icon");
  });

  it("uses the icon's default color when iconColor is omitted", () => {
    // `info-outline` defaults to ui-fleet-black-75 per InfoOutline.tsx.
    const { container } = render(<InfoBanner icon="info-outline" />);
    const path = container.querySelector(".info-banner__leading-icon path");
    expect(path?.getAttribute("fill")).toMatch(/ui-fleet-black-75/);
  });

  it("forwards iconColor to the leading Icon's fill", () => {
    const { container } = render(
      <InfoBanner icon="info-outline" iconColor="ui-fleet-black-50" />
    );
    const path = container.querySelector(".info-banner__leading-icon path");
    expect(path?.getAttribute("fill")).toMatch(/ui-fleet-black-50/);
  });
});
