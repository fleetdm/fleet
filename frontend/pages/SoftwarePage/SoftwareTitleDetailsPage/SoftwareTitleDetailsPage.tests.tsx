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
  const pageProps = ({
    router: createMockRouter(),
    routeParams: { id: String(titleId) },
    params: { id: String(titleId) },
    route: {},
    routes: [],
    location: {
      pathname: `/software/titles/${titleId}`,
      search: "?fleet_id=3",
      query: { fleet_id: "3" },
      hash: "",
    },
  } as unknown) as IPageProps;
  return render(<SoftwareTitleDetailsPage {...pageProps} />);
};

describe("SoftwareTitleDetailsPage", () => {
  beforeEach(() => {
    jest.spyOn(softwareAPI, "deleteSoftwareInstaller").mockResolvedValue({});
  });

  afterEach(() => jest.restoreAllMocks());

  it("deletes the in-house app of an iOS title without sending installer_id", async () => {
    const inHouseApp = createMockSoftwarePackage({
      installer_id: 10,
      name: "HelloWorld.ipa",
    });
    jest.spyOn(softwareAPI, "getSoftwareTitle").mockResolvedValue({
      software_title: createMockSoftwareTitleDetails({
        id: 1,
        source: "ios_apps",
        packages: [inHouseApp],
      }),
    });
    const { user } = renderPage(1);

    await user.click(await screen.findByText("HelloWorld.ipa"));
    await user.click(screen.getByRole("button", { name: "Delete software" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(softwareAPI.deleteSoftwareInstaller).toHaveBeenCalledWith(
        1,
        3,
        undefined,
        undefined
      )
    );
  });

  it("deletes a custom package of a macOS title with its installer_id", async () => {
    const customPackage = createMockSoftwarePackage({
      installer_id: 20,
      name: "Package.pkg",
    });
    jest.spyOn(softwareAPI, "getSoftwareTitle").mockResolvedValue({
      software_title: createMockSoftwareTitleDetails({
        id: 2,
        source: "apps",
        packages: [customPackage],
      }),
    });
    const { user } = renderPage(2);

    await user.click(await screen.findByText("Package.pkg"));
    await user.click(screen.getByRole("button", { name: "Delete version" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(softwareAPI.deleteSoftwareInstaller).toHaveBeenCalledWith(
        2,
        3,
        20,
        undefined
      )
    );
  });
});
