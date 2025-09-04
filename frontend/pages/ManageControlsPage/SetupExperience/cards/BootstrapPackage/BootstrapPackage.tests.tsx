import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  createSetupExperienceBootstrapPackageHandler,
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

import BootstrapPackage from "./BootstrapPackage";

/**
 * sets up some default backend mocks for the tests. Override what you need
 * with mockServer.use() in the test itself.
 */
const setuDefaultBackendMocks = () => {
  mockServer.use(createGetConfigHandler());

  // default is no run script or install software already added
  mockServer.use(errorNoSetupExperienceScriptHandler);
  mockServer.use(createSetupExperienceSoftwareHandler());

  // default will be a bootstrap package already uploaded
  mockServer.use(
    createSetupExperienceBootstrapPackageHandler({ name: "foo-package.pkg" })
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
  it("renders the status table and bootstrap package if a package has been uploaded", async () => {
    setuDefaultBackendMocks();

    const render = createCustomRenderer({ withBackendMock: true });
    render(<BootstrapPackage currentTeamId={0} />);

    await screen.findByText(/status/gi);

    // table is showing
    expect(screen.getByText("Status")).toBeVisible();
    expect(screen.getByText("Hosts")).toBeVisible();

    // bootstrap package preview is showing
    expect(screen.getByText("foo-package.pkg")).toBeVisible();
  });

  it("render the bootstrap package uploader if a package has not been uploaded", async () => {
    setuDefaultBackendMocks();
    mockServer.use(errorNoBootstrapPackageMetadataHandler);

    const render = createCustomRenderer({ withBackendMock: true });
    render(<BootstrapPackage currentTeamId={0} />);

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
    setuDefaultBackendMocks();
    mockServer.use(errorNoBootstrapPackageMetadataHandler);

    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<BootstrapPackage currentTeamId={0} />);

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders the advanced options as disabled if there are already added install software", async () => {
    setuDefaultBackendMocks();
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

    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<BootstrapPackage currentTeamId={0} />);

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders the advanced options as disabled if there is alreaddy a run script added", async () => {
    setuDefaultBackendMocks();
    mockServer.use(createSetupExperienceScriptHandler());

    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<BootstrapPackage currentTeamId={0} />);

    await screen.findByText("Show advanced options");
    await user.click(screen.getByText("Show advanced options"));

    expect(
      screen.getByLabelText("Install Fleet's agent (fleetd) manually")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});
