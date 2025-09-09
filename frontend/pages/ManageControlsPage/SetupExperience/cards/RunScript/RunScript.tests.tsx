import React from "react";
import { screen } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import {
  createSetupExperienceScriptHandler,
  errorNoSetupExperienceScriptHandler,
} from "test/handlers/setup-experience-handlers";

import RunScript from "./RunScript";

describe("RunScript", () => {
  it("should render the 'turn on automatic enrollment' message when MDM isn't configured", async () => {
    mockServer.use(errorNoSetupExperienceScriptHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { config: { mdm: { enabled_and_configured: false } } } },
    });

    render(<RunScript router={createMockRouter()} currentTeamId={1} />);

    expect(
      await screen.getByText(/turn on automatic enrollment/)
    ).toBeInTheDocument();
  });
  it("should render the script uploader when no script has been uploaded", async () => {
    mockServer.use(errorNoSetupExperienceScriptHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          config: {
            mdm: {
              enabled_and_configured: true,
              apple_bm_enabled_and_configured: true,
            },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    render(<RunScript router={createMockRouter()} currentTeamId={1} />);

    expect(await screen.findByRole("button", { name: "Upload" })).toBeVisible();
  });

  it("should render the uploaded script uploader when a script has been uploaded", async () => {
    mockServer.use(createSetupExperienceScriptHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          config: {
            mdm: {
              enabled_and_configured: true,
              apple_bm_enabled_and_configured: true,
            },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    render(<RunScript router={createMockRouter()} currentTeamId={1} />);

    expect(
      await screen.findByText("Script will run during setup:")
    ).toBeVisible();
    expect(await screen.findByText("Test Script.sh")).toBeVisible();
  });
});
