import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  createSetupExperienceBootstrapMetadataHandler,
  createSetupExperienceScriptHandler,
  createSetupExperienceSoftwareHandler,
  createSetuUpExperienceBootstrapSummaryHandler,
  errorNoBootstrapPackageMetadataHandler,
  errorNoSetupExperienceScriptHandler,
} from "test/handlers/setup-experience-handlers";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import { createMockMdmConfig } from "__mocks__/configMock";

import BootstrapPackage from "./BootstrapPackage";

/**
 * sets up some default backend mocks for the tests. Override what you need
 * with mockServer.use() in the test itself.
 */
const setupDefaultBackendMocks = () => {
  mockServer.use(createGetConfigHandler());

  // default is no run script or install software already added
  mockServer.use(errorNoSetupExperienceScriptHandler);
  mockServer.use(createSetupExperienceSoftwareHandler());

  // default will be a bootstrap package already uploaded
  mockServer.use(
    createSetupExperienceBootstrapMetadataHandler({ name: "foo-package.pkg" })
  );
  mockServer.use(
    createSetuUpExperienceBootstrapSummaryHandler({
      installed: 1,
      pending: 2,
      failed: 3,
    })
  );
};

describe("BootstrapPackage", () => {
  it("renders the 'turn on automatic enrollment' message when MDM isn't configured", async () => {
    setupDefaultBackendMocks();
    mockServer.use(
      createGetConfigHandler({
        mdm: createMockMdmConfig({ enabled_and_configured: false }),
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<BootstrapPackage router={createMockRouter()} currentTeamId={0} />);

    await waitFor(() => {
      expect(
        screen.getByText(/turn on automatic enrollment/)
      ).toBeInTheDocument();
    });
  });
  it("renders the 'turn on automatic enrollment' message when MDM is configured, but ABM is not", async () => {
    setupDefaultBackendMocks();
    mockServer.use(
      createGetConfigHandler({
        mdm: createMockMdmConfig({
          enabled_and_configured: true,
          apple_bm_enabled_and_configured: false,
        }),
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<BootstrapPackage router={createMockRouter()} currentTeamId={0} />);

    await waitFor(() => {
      expect(
        screen.getByText(/turn on automatic enrollment/)
      ).toBeInTheDocument();
    });
  });
  it("renders the status table and bootstrap package if a package has been uploaded", async () => {
    setupDefaultBackendMocks();

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<BootstrapPackage router={createMockRouter()} currentTeamId={0} />);

    await screen.findByText(/status/gi);

    // table is showing
    expect(screen.getByText("Status")).toBeVisible();
    expect(screen.getByText("Hosts")).toBeVisible();

    // bootstrap package preview is showing
    expect(screen.getByText("foo-package.pkg")).toBeVisible();
  });

  it("render the bootstrap package uploader if a package has not been uploaded", async () => {
    setupDefaultBackendMocks();
    mockServer.use(errorNoBootstrapPackageMetadataHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<BootstrapPackage router={createMockRouter()} currentTeamId={0} />);

    await screen.findByText(/Upload a bootstrap package/gi);

    // upload description is showing
    expect(
      screen.getByText(
        /Upload a bootstrap package to install a configuration/gi
      )
    ).toBeVisible();

    // uploader is showing
    expect(screen.getByText("Package (.pkg)")).toBeVisible();
    expect(screen.getByText("Upload")).toBeVisible();
  });

  it("renders the advanced options as disabled if there is no bootstrap package uploaded", async () => {
    setupDefaultBackendMocks();
    mockServer.use(errorNoBootstrapPackageMetadataHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(
      <BootstrapPackage router={createMockRouter()} currentTeamId={0} />
    );

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders the advanced options as disabled if there are already added install software", async () => {
    setupDefaultBackendMocks();
    mockServer.use(
      createSetupExperienceSoftwareHandler({
        software_titles: [
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
        ],
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(
      <BootstrapPackage router={createMockRouter()} currentTeamId={0} />
    );

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders the advanced options as disabled if there is alreaddy a run script added", async () => {
    setupDefaultBackendMocks();
    mockServer.use(createSetupExperienceScriptHandler());

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(
      <BootstrapPackage router={createMockRouter()} currentTeamId={0} />
    );

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});
