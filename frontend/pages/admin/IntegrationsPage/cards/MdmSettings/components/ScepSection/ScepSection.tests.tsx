import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import ScepSection from "./ScepSection";

describe("Scep Section", () => {
  it("renders mdm is off message when apple mdm is not turned on ", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: false }),
          }),
        },
      },
    });

    render(
      <ScepSection router={createMockRouter()} isScepOn={false} isPremiumTier />
    );

    expect(
      await screen.findByText(/first turn on Apple \(macOS, iOS, iPadOS\) MDM/i)
    ).toBeInTheDocument();
  });

  it("renders add scep when scep is disabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
    });

    render(
      <ScepSection router={createMockRouter()} isScepOn={false} isPremiumTier />
    );

    expect(
      await screen.findByRole("button", { name: "Add SCEP" })
    ).toBeInTheDocument();
  });

  it("renders edit scep when scep is enabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
    });
    render(<ScepSection router={createMockRouter()} isScepOn isPremiumTier />);
    expect(
      await screen.findByRole("button", { name: "Edit" })
    ).toBeInTheDocument();
  });

  it("render the premium message when not in premium tier", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
    });
    render(
      <ScepSection
        router={createMockRouter()}
        isScepOn={false}
        isPremiumTier={false}
      />
    );
    expect(
      await screen.findByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
  });
});
