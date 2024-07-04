import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  defaultVppInfoHandler,
  errorNoVppInfoHandler,
} from "test/handlers/apple_mdm";
import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import Vpp from "./Vpp";

describe("Vpp Section", () => {
  it("render turn on apple mdm message when apple mdm is not turned on ", async () => {
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

    render(<Vpp router={createMockRouter()} />);

    expect(
      await screen.findByRole("button", { name: "Turn on macOS MDM" })
    ).toBeInTheDocument();
  });

  it("render enable vpp when vpp is disabled", async () => {
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

    render(<Vpp router={createMockRouter()} />);

    expect(
      await screen.findByRole("button", { name: "Enable" })
    ).toBeInTheDocument();
  });

  it("render edit vpp when vpp is enabled", async () => {
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
    render(<Vpp router={createMockRouter()} />);
    expect(
      await screen.findByRole("button", { name: "Edit" })
    ).toBeInTheDocument();
  });
});
