import React from "react";
import { screen, waitFor } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { createSetupExperienceSoftwareHandler } from "test/handlers/setup-experience-handlers";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import { createGetTeamHandler } from "test/handlers/team-handlers";
import { createMockMdmConfig } from "__mocks__/configMock";

import InstallSoftware from "./InstallSoftware";

describe("InstallSoftware", () => {
  it("renders the page description on the empty state when MDM isn't configured", async () => {
    mockServer.use(createSetupExperienceSoftwareHandler());
    mockServer.use(
      createGetConfigHandler({
        mdm: createMockMdmConfig({ enabled_and_configured: false }),
      })
    );
    mockServer.use(createGetTeamHandler({}));
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <InstallSoftware
        router={createMockRouter()}
        currentTeamId={1}
        urlPlatformParam="macos"
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(/Turn on MDM and automatic enrollment/)
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Install software on hosts that automatically enroll/)
    ).toBeVisible();
  });
});
