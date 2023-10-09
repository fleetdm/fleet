import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import ProfileStatusIndicator from "./ProfileStatusIndicator";

describe("ProfileStatusIndicator component", () => {
  it("Renders the text and icon", () => {
    const indicatorText = "test text";
    render(
      <ProfileStatusIndicator
        indicatorText={indicatorText}
        iconName="success"
      />
    );
    const renderedIndicatorText = screen.getByText(indicatorText);
    const renderedIcon = screen.getByTestId("success-icon");

    expect(renderedIndicatorText).toBeInTheDocument();
    expect(renderedIcon).toBeInTheDocument();
  });

  it("Renders text, icon, and tooltip", () => {
    const indicatorText = "test text";
    const tooltipText = "test tooltip text";
    render(
      <ProfileStatusIndicator
        indicatorText={indicatorText}
        iconName="success"
        tooltip={{ tooltipText }}
      />
    );
    const renderedIndicatorText = screen.getByText(indicatorText);
    const renderedIcon = screen.getByTestId("success-icon");
    const renderedTooltipText = screen.getByText(tooltipText);

    expect(renderedIndicatorText).toBeInTheDocument();
    expect(renderedIcon).toBeInTheDocument();
    expect(renderedTooltipText).toBeInTheDocument();
  });

  it("Renders text, icon, and onClick", () => {
    const indicatorText = "test text";
    const onClick = () => {
      const newDiv = document.createElement("div");
      newDiv.appendChild(document.createTextNode("onClick called"));
      document.body.appendChild(newDiv);
    };
    render(
      <ProfileStatusIndicator
        indicatorText={indicatorText}
        iconName="success"
        onClick={() => {
          onClick();
        }}
      />
    );

    const renderedIndicatorText = screen.getByText(indicatorText);
    const renderedIcon = screen.getByTestId("success-icon");
    const renderedButton = screen.getByRole("button");

    expect(renderedIndicatorText).toBeInTheDocument();
    expect(renderedIcon).toBeInTheDocument();
    expect(renderedButton).toBeInTheDocument();

    fireEvent.click(renderedButton);
    expect(screen.getByText("onClick called")).toBeInTheDocument();
  });
});
