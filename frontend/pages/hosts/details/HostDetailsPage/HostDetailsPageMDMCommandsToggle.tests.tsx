import React from "react";
import { screen, waitFor } from "@testing-library/react";

import createMockHost from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import { IHost } from "interfaces/host";
import hostAPI from "services/entities/hosts";
import activitiesAPI from "services/entities/activities";
import teamAPI from "services/entities/teams";
import commandAPI from "services/entities/command";

import HostDetailsPage from "./HostDetailsPage";

jest.mock("services/entities/hosts");
jest.mock("services/entities/activities");
jest.mock("services/entities/teams");
jest.mock("services/entities/command");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const mockLocation = {
  pathname: "/hosts/1",
  query: {},
  search: "",
};

const ADMIN = createMockUser();

const mockAppleHost = (): IHost => {
  const host = createMockHost({ platform: "darwin", status: "online" });
  host.mdm.enrollment_status = "On (manual)";
  host.mdm.connected_to_fleet = true;
  return host;
};

const stubQueries = (host: IHost) => {
  (hostAPI.loadHostDetails as jest.Mock).mockResolvedValue({ host });
  (hostAPI.loadHostDetailsExtension as jest.Mock).mockResolvedValue({
    macadmins: null,
  });
  (hostAPI.getHostCertificates as jest.Mock).mockResolvedValue({
    certificates: [],
    meta: { has_next_results: false, has_previous_results: false },
  });
  (activitiesAPI.getHostPastActivities as jest.Mock).mockResolvedValue({
    activities: [],
    meta: { has_next_results: false, has_previous_results: false },
  });
  (activitiesAPI.getHostUpcomingActivities as jest.Mock).mockResolvedValue({
    activities: [],
    count: 0,
    meta: { has_next_results: false, has_previous_results: false },
  });
  (teamAPI.loadAll as jest.Mock).mockResolvedValue({ teams: [] });
  (commandAPI.getCommands as jest.Mock).mockResolvedValue({
    results: [],
    count: 0,
    meta: { has_next_results: false, has_previous_results: false },
  });
};

/** The toggle only renders when Apple MDM is on, which the page reads from
 * context rather than deriving from the config. */
const renderPage = () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser: ADMIN,
        isGlobalAdmin: true,
        isPremiumTier: true,
        isMacMdmEnabledAndConfigured: true,
        config: createMockConfig(),
      },
    },
  });

  return render(
    <HostDetailsPage
      router={createMockRouter()}
      location={mockLocation}
      params={{ host_id: "1" }}
    />
  );
};

describe("HostDetailsPage - Show MDM commands toggle", () => {
  afterEach(() => {
    localStorage.clear();
  });

  it("keeps the toggle on across a remount", async () => {
    stubQueries(mockAppleHost());

    const { user, unmount } = renderPage();
    // The toggle is a switch whose label sits beside it, and one renders per
    // activity tab, so take the first.
    const [toggle] = await screen.findAllByRole("switch");
    expect(toggle).not.toBeChecked();
    await user.click(toggle);

    expect(localStorage.getItem("FLEET::hostDetailsShowMDMCommands")).toBe(
      "true"
    );
    unmount();

    // Remounting stands in for the page refresh that used to drop the choice.
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByRole("switch")[0]).toBeChecked();
    });
  });
});
