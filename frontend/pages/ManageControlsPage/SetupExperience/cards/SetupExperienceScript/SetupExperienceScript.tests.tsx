import React from "react";
import { screen } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";
import {
  defaultSetupExperienceScriptHandler,
  errorNoSetupExperienceScript,
} from "test/handlers/setup-experience-handlers";

import SetupExperienceScript from "./SetupExperienceScript";

describe("SetupExperienceScript", () => {
  it("should render the script uploader when no script has been uploaded", async () => {
    mockServer.use(errorNoSetupExperienceScript);
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SetupExperienceScript currentTeamId={1} />);

    expect(await screen.findByRole("button", { name: "Upload" })).toBeVisible();
  });

  it("should render the uploaded script uploader when a script has been uploaded", async () => {
    mockServer.use(defaultSetupExperienceScriptHandler);
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SetupExperienceScript currentTeamId={1} />);

    expect(
      await screen.findByText("Script will run during setup:")
    ).toBeVisible();
    expect(await screen.findByText("Test Script.sh")).toBeVisible();
  });
});
