import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockMdmConfig } from "__mocks__/configMock";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import { createSetupExperienceSoftwareHandler } from "test/handlers/setup-experience-handlers";
import { createGetTeamHandler } from "test/handlers/team-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import InstallSoftware from "./InstallSoftware";

const setupMdmNotConfigured = () => {
  mockServer.use(createSetupExperienceSoftwareHandler());
  mockServer.use(
    createGetConfigHandler({
      mdm: createMockMdmConfig({ enabled_and_configured: false }),
    })
  );
  mockServer.use(createGetTeamHandler({}));
};

const setupMdmConfigured = () => {
  mockServer.use(createSetupExperienceSoftwareHandler());
  mockServer.use(createGetConfigHandler());
  mockServer.use(createGetTeamHandler({}));
};

describe("InstallSoftware", () => {
  it("renders the page description on the empty state when MDM isn't configured", async () => {
    setupMdmNotConfigured();
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

  it("renders the software form when MDM is configured", async () => {
    setupMdmConfigured();
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

    // Page description is always visible
    expect(
      screen.getByText(/Install software on hosts that automatically enroll/)
    ).toBeVisible();

    // The form renders (Save button appears)
    expect(await screen.findByRole("button", { name: "Save" })).toBeVisible();
  });

  it("renders the Android empty state with correct messaging", async () => {
    mockServer.use(createSetupExperienceSoftwareHandler());
    mockServer.use(
      createGetConfigHandler({
        mdm: createMockMdmConfig({ android_enabled_and_configured: false }),
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
        urlPlatformParam="android"
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/Turn on Android MDM/)).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Install software on hosts that enroll to Fleet/)
    ).toBeVisible();
  });

  describe("versioned App Store app helper suffix", () => {
    it.each(["ios", "ipados", "android"] as const)(
      "appends the first-added disclaimer on the %s tab when MDM is configured",
      async (platform) => {
        mockServer.use(createSetupExperienceSoftwareHandler());
        // Default: Apple MDM on, Android MDM off; flip Android on so all three tabs render.
        mockServer.use(
          createGetConfigHandler({
            mdm: createMockMdmConfig({ android_enabled_and_configured: true }),
          })
        );
        mockServer.use(createGetTeamHandler({}));
        const render = createCustomRenderer({ withBackendMock: true });

        render(
          <InstallSoftware
            router={createMockRouter()}
            currentTeamId={1}
            urlPlatformParam={platform}
          />
        );

        expect(
          await screen.findByText(
            /so first added version will be always installed/i
          )
        ).toBeVisible();
      }
    );

    it("omits the suffix on macOS, Windows, and Linux tabs", async () => {
      setupMdmConfigured();
      const render = createCustomRenderer({ withBackendMock: true });

      render(
        <InstallSoftware
          router={createMockRouter()}
          currentTeamId={1}
          urlPlatformParam="macos"
        />
      );

      expect(await screen.findByRole("button", { name: "Save" })).toBeVisible();
      expect(
        screen.queryByText(/so first added version will be always installed/i)
      ).toBeNull();
    });
  });
});
