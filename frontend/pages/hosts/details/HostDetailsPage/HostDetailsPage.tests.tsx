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
import { notify } from "components/ToastNotification";

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

const mockPendingWindowsHost = (status: "online" | "offline"): IHost => {
  const host = createMockHost({
    platform: "windows",
    status,
    refetch_requested: true,
    last_enrolled_at: "2000-01-01T00:00:00Z",
  });
  host.mdm.enrollment_status = "Pending";
  return host;
};

/** An Apple host that is MDM-enrolled and online -- the only combination that
 * pings APNS alongside the refetch. */
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

  it("pings APNS alongside the refetch for observer and above", async () => {
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

    // Global observer: fires the ping as well.
    const { user: observer } = renderPageAs(OBSERVER, false);
    await observer.click(
      await screen.findByRole("button", { name: /refetch/i })
    );
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(1);
  });
});

describe("HostDetailsPage - pending hosts", () => {
  afterEach(() => {
    jest.resetAllMocks();
  });

  it("doesn't spin on vitals or report the host offline", async () => {
    stubQueries(mockPendingWindowsHost("online"));
    // The host row ages out of the 60-second online window while the page is open.
    (hostAPI.loadHostDetails as jest.Mock)
      .mockResolvedValueOnce({ host: mockPendingWindowsHost("online") })
      .mockResolvedValue({ host: mockPendingWindowsHost("offline") });

    renderPageAs(ADMIN, true);
    await screen.findByText("Vitals");
    // Give the poll timer (2s) room to fire if it was scheduled.
    await new Promise((resolve) => setTimeout(resolve, 3000));

    expect(
      screen.queryByText(/fetching fresh vitals/i)
    ).not.toBeInTheDocument();
    expect(notify.error).not.toHaveBeenCalled();
  }, 15000);

  it("still reports back when the user asks for the refetch", async () => {
    stubQueries(mockPendingWindowsHost("online"));
    (hostAPI.loadHostDetails as jest.Mock)
      .mockResolvedValueOnce({ host: mockPendingWindowsHost("online") })
      .mockResolvedValue({ host: mockPendingWindowsHost("offline") });

    const { user } = renderPageAs(ADMIN, true);
    await user.click(await screen.findByRole("button", { name: /refetch/i }));
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });

    await waitFor(
      () => {
        expect(notify.error).toHaveBeenCalledWith(
          "This host is offline. Please try refetching host vitals later."
        );
      },
      { timeout: 10000 }
    );
  }, 20000);
});
