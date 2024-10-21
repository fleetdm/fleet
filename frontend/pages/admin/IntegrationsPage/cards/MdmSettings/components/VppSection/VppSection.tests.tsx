import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  defaultVppInfoHandler,
  errorNoVppInfoHandler,
} from "test/handlers/apple_mdm";
import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import VppSection from "./VppSection";

describe("Vpp Section", () => {
  it("renders mdm is off message when apple mdm is not turned on ", async () => {
    mockServer.use(defaultVppInfoHandler);

    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: false }),
          }),
        },
      },
      withBackendMock: true,
    });

    render(
      <VppSection router={createMockRouter()} isVppOn={false} isPremiumTier />
    );

    expect(
      await screen.findByText(
        "To enable Volume Purchasing Program (VPP), first turn on Apple (macOS, iOS, iPadOS) MDM."
      )
    ).toBeInTheDocument();
  });

  it("renders add vpp when vpp is disabled", async () => {
    mockServer.use(errorNoVppInfoHandler);

    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
      withBackendMock: true,
    });

    render(
      <VppSection router={createMockRouter()} isVppOn={false} isPremiumTier />
    );

    expect(
      await screen.findByRole("button", { name: "Add VPP" })
    ).toBeInTheDocument();
  });

  it("renders edit vpp when vpp is enabled", async () => {
    mockServer.use(defaultVppInfoHandler);

    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
      withBackendMock: true,
    });
    render(<VppSection router={createMockRouter()} isVppOn isPremiumTier />);
    expect(
      await screen.findByRole("button", { name: "Edit" })
    ).toBeInTheDocument();
  });

  it("render the premium message when not in premium tier", async () => {
    mockServer.use(defaultVppInfoHandler);

    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
      withBackendMock: true,
    });
    render(
      <VppSection
        router={createMockRouter()}
        isVppOn={false}
        isPremiumTier={false}
      />
    );
    expect(
      await screen.findByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
  });
});
