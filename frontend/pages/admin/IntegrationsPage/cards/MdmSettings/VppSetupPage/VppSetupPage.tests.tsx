import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  defaultVppInfoHandler,
  errorNoVppInfoHandler,
} from "test/handlers/apple_mdm";

import VppSetupPage from "./VppSetupPage";

describe("VppSetupPage", () => {
  it("renders the VPP setup steps content when VPP is not set up", async () => {
    mockServer.use(errorNoVppInfoHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<VppSetupPage router={createMockRouter()} />);

    // This is part of the setup steps content we expect to see.
    expect(await screen.findByText(/Sign in to/g)).toBeInTheDocument();
    // This is the upload token UI we expect to see.
    expect(
      await screen.findByRole("button", { name: "Upload" })
    ).toBeInTheDocument();
  });

  it("renders the VPP disable and renew content when VPP is set up", async () => {
    mockServer.use(defaultVppInfoHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<VppSetupPage router={createMockRouter()} />);

    expect(await screen.findByText("Organization name")).toBeInTheDocument();

    expect(
      await screen.findByRole("button", { name: "Disable" })
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Renew token" })
    ).toBeInTheDocument();
  });
});
