import React from "react";
import { render, screen } from "@testing-library/react";

import SelfServicePreview from "./SelfServicePreview";

describe("SelfServicePreview", () => {
  it("renders mobile preview with screenshot, name, version, and icon", () => {
    const MockIcon = () => <div>Mock icon</div>;

    render(
      <SelfServicePreview
        isIosOrIpadosApp
        contactUrl="https://example.com/help"
        name="App name"
        displayName="Display name"
        versionLabel="1.2.3"
        renderIcon={() => <MockIcon />}
      />
    );

    expect(
      screen.getByAltText("Preview icon on Fleet Desktop > Self-service")
    ).toBeVisible();

    expect(screen.getByText("Mock icon")).toBeVisible();
    expect(screen.getByText("Display name")).toBeVisible();
    expect(screen.getByText("1.2.3")).toBeVisible();
  });

  it("falls back to name when displayName is empty in mobile preview", () => {
    render(
      <SelfServicePreview
        isIosOrIpadosApp
        contactUrl="https://example.com/help"
        name="Fallback name"
        displayName=""
        versionLabel="1.2.3"
        renderIcon={() => <div>Icon</div>}
      />
    );

    expect(screen.getByText("Fallback name")).toBeVisible();
  });

  it("renders desktop preview with header, search field, categories menu, and table", () => {
    const MockTable = () => <div>Mock table</div>;

    render(
      <SelfServicePreview
        isIosOrIpadosApp={false}
        contactUrl="https://example.com/help"
        name="App name"
        displayName="Display name"
        versionLabel="1.2.3"
        renderIcon={() => <div>Icon</div>}
        renderTable={() => <MockTable />}
      />
    );

    expect(screen.getByText(/Self-service/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Search by name")).toBeInTheDocument();
    expect(screen.getByText(/Browsers/i)).toBeInTheDocument();
    expect(screen.getByText("Mock table")).toBeVisible();
  });
});
