import React from "react";
import { screen, waitFor } from "@testing-library/react";

import createMockHost from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import { IHost } from "interfaces/host";
import { IUser } from "interfaces/user";
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
const OBSERVER = createMockUser({ role: "observer", global_role: "observer" });

/** An Apple host that is MDM-enrolled and online -- the only combination that
 * pings APNS alongside the refetch. */
const mockAppleHost = (): IHost => {
  const host = createMockHost({ platform: "darwin", status: "online" });
  host.mdm.enrollment_status = "On (manual)";
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
  (hostAPI.refetch as jest.Mock).mockResolvedValue({});
  (hostAPI.apnsPing as jest.Mock).mockResolvedValue({});
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

const renderPageAs = (currentUser: IUser, isGlobalAdmin: boolean) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser,
        isGlobalAdmin,
        isPremiumTier: true,
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

describe("HostDetailsPage - APNS ping on refetch", () => {
  afterEach(() => {
    jest.resetAllMocks();
  });

  it("only pings APNS alongside the refetch for a maintainer or higher", async () => {
    stubQueries(mockAppleHost());

    // Global admin: refetch fires the ping too.
    const { user, unmount } = renderPageAs(ADMIN, true);
    await user.click(await screen.findByRole("button", { name: /refetch/i }));
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(1);
    unmount();

    (hostAPI.refetch as jest.Mock).mockClear();
    (hostAPI.apnsPing as jest.Mock).mockClear();

    // Global observer: the refetch still goes out, the ping does not.
    const { user: observer } = renderPageAs(OBSERVER, false);
    await observer.click(
      await screen.findByRole("button", { name: /refetch/i })
    );
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).not.toHaveBeenCalled();
  });
});
