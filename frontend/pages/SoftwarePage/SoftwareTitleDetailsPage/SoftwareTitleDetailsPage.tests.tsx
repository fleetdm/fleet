import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import {
  createMockSoftwarePackage,
  createMockSoftwareTitleDetails,
} from "__mocks__/softwareMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import softwareAPI from "services/entities/software";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareTitleDetailsPage from "./SoftwareTitleDetailsPage";

type IPageProps = React.ComponentProps<typeof SoftwareTitleDetailsPage>;

const renderPage = (titleId: number) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        currentUser: createMockUser({ global_role: "admin" }),
        availableTeams: [
          createMockTeamSummary({ id: 3, name: "Workstations" }),
        ],
        setCurrentTeam: jest.fn(),
        config: createMockConfig(),
      },
    },
  });
  return render(
    <SoftwareTitleDetailsPage
      router={createMockRouter()}
      routeParams={{ id: String(titleId) }}
      location={
        ({
          pathname: `/software/titles/${titleId}`,
          search: "?fleet_id=3",
          query: { fleet_id: "3" },
          hash: "",
        } as unknown) as IPageProps["location"]
      }
    />
  );
};

describe("SoftwareTitleDetailsPage", () => {
  beforeEach(() => {
    jest.spyOn(softwareAPI, "deleteSoftwareInstaller").mockResolvedValue({});
  });

  afterEach(() => jest.restoreAllMocks());

  it("deletes the in-house app of an iOS title without sending installer_id", async () => {
    const inHouseApp = createMockSoftwarePackage({
      installer_id: 182,
      name: "HelloWorld.ipa",
    });
    jest.spyOn(softwareAPI, "getSoftwareTitle").mockResolvedValue({
      software_title: createMockSoftwareTitleDetails({
        id: 2295,
        source: "ios_apps",
        software_package: inHouseApp,
        packages: [inHouseApp],
      }),
    });
    const { user } = renderPage(2295);

    await user.click(await screen.findByText("HelloWorld.ipa"));
    await user.click(screen.getByRole("button", { name: "Delete software" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(softwareAPI.deleteSoftwareInstaller).toHaveBeenCalledWith(
        2295,
        3,
        undefined,
        undefined
      )
    );
  });

  it("deletes a custom package of a macOS title with its installer_id", async () => {
    const customPackage = createMockSoftwarePackage({
      installer_id: 3248,
      name: "Package.pkg",
    });
    jest.spyOn(softwareAPI, "getSoftwareTitle").mockResolvedValue({
      software_title: createMockSoftwareTitleDetails({
        id: 6809,
        source: "apps",
        software_package: customPackage,
        packages: [customPackage],
      }),
    });
    const { user } = renderPage(6809);

    await user.click(await screen.findByText("Package.pkg"));
    await user.click(screen.getByRole("button", { name: "Delete version" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(softwareAPI.deleteSoftwareInstaller).toHaveBeenCalledWith(
        6809,
        3,
        3248,
        undefined
      )
    );
  });
});
