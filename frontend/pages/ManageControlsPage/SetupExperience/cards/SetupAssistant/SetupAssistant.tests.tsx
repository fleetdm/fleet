import React from "react";
import { screen, waitFor } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import { createGetTeamHandler } from "test/handlers/team-handlers";
import { createMockMdmConfig } from "__mocks__/configMock";

import SetupAssistant from "./SetupAssistant";

describe("SetupAssistant", () => {
  it("renders the page description on the empty state when MDM isn't configured", async () => {
    mockServer.use(
      createGetConfigHandler({
        mdm: createMockMdmConfig({ enabled_and_configured: false }),
      })
    );
    mockServer.use(createGetTeamHandler({}));
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<SetupAssistant router={createMockRouter()} currentTeamId={1} />);

    await waitFor(() => {
      expect(
        screen.getByText(/first turn on automatic enrollment/)
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Add an automatic enrollment profile/)
    ).toBeVisible();
  });
});
