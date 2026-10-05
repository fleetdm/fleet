import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { createMockActivity } from "__mocks__/activityMock";
import createMockConfig from "__mocks__/configMock";
import createMockHost from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
import { ActivityType } from "interfaces/activity";
import { IHost } from "interfaces/host";
import { IUser } from "interfaces/user";
import activitiesAPI from "services/entities/activities";
import commandAPI from "services/entities/command";
import hostAPI from "services/entities/hosts";
import teamAPI from "services/entities/teams";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import local from "utilities/local";

import HostDetailsPage, {
  getMDMCommandsToggleLocalState,
  setMDMCommandsToggleLocalState,
} from "./HostDetailsPage";

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

// The server's "never" sentinel for timestamps that have not been set yet.
const NEVER = "2000-01-01T00:00:00Z";

const mockPendingWindowsHost = (status: "online" | "offline"): IHost => {
  const host = createMockHost({
    platform: "windows",
    status,
    refetch_requested: true,
    last_enrolled_at: "2000-01-01T00:00:00Z",
    detail_updated_at: NEVER,
  });
  host.mdm.enrollment_status = "Pending";
  return host;
};

/** A host whose agent has enrolled but has not reported vitals yet, e.g. while setup experience is running. */
const mockNeverFetchedWindowsHost = (status: "online" | "offline"): IHost =>
  createMockHost({
    platform: "windows",
    status,
    refetch_requested: true,
    last_enrolled_at: "2026-09-23T00:00:00Z",
    detail_updated_at: NEVER,
  });

/** An Apple host that is MDM-enrolled and online -- the only combination that
 * pings APNS alongside the refetch. */
const mockAppleHost = (): IHost => {
  const host = createMockHost({ platform: "darwin", status: "online" });
  host.mdm.enrollment_status = "On (manual)";
  host.mdm.connected_to_fleet = true;
  return host;
};

const mockWindowsHost = (): IHost => {
  const host = createMockHost({ platform: "windows", status: "online" });
  host.mdm.enrollment_status = null;
  host.mdm.connected_to_fleet = false;
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

const renderHostDetails = (overrides?: {
  currentUser?: IUser;
  isGlobalAdmin?: boolean;
  isMacMdmEnabledAndConfigured?: boolean;
  location?: typeof mockLocation;
}) => {
  const {
    currentUser = ADMIN,
    isGlobalAdmin = true,
    isMacMdmEnabledAndConfigured = false,
    location = mockLocation,
  } = overrides || {};

  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser,
        isGlobalAdmin,
        isPremiumTier: true,
        isMacMdmEnabledAndConfigured,
        config: createMockConfig(),
      },
    },
  });

  return render(
    <HostDetailsPage
      router={createMockRouter()}
      location={location}
      params={{ host_id: "1" }}
    />
  );
};

beforeEach(() => {
  class MockResizeObserver {
    observe = jest.fn();
    unobserve = jest.fn();
    disconnect = jest.fn();
  }

  global.ResizeObserver = MockResizeObserver as typeof ResizeObserver;
});

describe("HostDetailsPage - APNS ping on refetch", () => {
  afterEach(() => {
    local.clear();
    jest.resetAllMocks();
  });

  it("pings APNS alongside the refetch for observer and above", async () => {
    stubQueries(mockAppleHost());

    // Global admin: refetch fires the ping too.
    const { user, unmount } = renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });
    await user.click(await screen.findByRole("button", { name: /refetch/i }));
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(1);
    unmount();

    (hostAPI.refetch as jest.Mock).mockClear();
    (hostAPI.apnsPing as jest.Mock).mockClear();

    // Global observer: fires the ping as well.
    const { user: observer } = renderHostDetails({
      currentUser: OBSERVER,
      isGlobalAdmin: false,
    });
    await observer.click(
      await screen.findByRole("button", { name: /refetch/i })
    );
    await waitFor(() => {
      expect(hostAPI.refetch).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(1);
  });
});

describe("HostDetailsPage - MDM status modal Check in now", () => {
  afterEach(() => {
    jest.useRealTimers();
    jest.resetAllMocks();
  });

  it("delays the host details refetch by 5 seconds after a successful check-in", async () => {
    stubQueries(mockAppleHost());
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      host_dep_assignment: null,
    });

    renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
      location: { ...mockLocation, query: { show_mdm_status: "true" } },
    });

    const checkInButton = await screen.findByRole("button", {
      name: /check in now/i,
    });
    const callsBeforeCheckIn = (hostAPI.loadHostDetails as jest.Mock).mock.calls
      .length;

    // Fake timers only from here on, so user-event can control its own
    // internal delays via the advanceTimers option.
    jest.useFakeTimers();
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });

    await user.click(checkInButton);
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(1);

    // No immediate refetch -- the app delays it 5s so the device has time to check in.
    expect((hostAPI.loadHostDetails as jest.Mock).mock.calls.length).toBe(
      callsBeforeCheckIn
    );

    await act(async () => {
      await jest.advanceTimersByTimeAsync(5000);
    });

    expect(
      (hostAPI.loadHostDetails as jest.Mock).mock.calls.length
    ).toBeGreaterThan(callsBeforeCheckIn);
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

    renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });
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

    const { user } = renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });
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

describe("HostDetailsPage - hosts that haven't reported vitals", () => {
  const realNow = Date.now;
  let elapsedMs = 0;
  let dateNowSpy: jest.SpyInstance;

  beforeEach(() => {
    elapsedMs = 0;
    dateNowSpy = jest
      .spyOn(Date, "now")
      .mockImplementation(() => realNow() + elapsedMs);
  });

  afterEach(() => {
    dateNowSpy.mockRestore();
    jest.resetAllMocks();
  });

  it.each([
    {
      name: "drops out of the online window",
      laterStatus: "offline",
      pollMs: 0,
    },
    { name: "outlasts the poll window", laterStatus: "online", pollMs: 61000 },
  ] as const)(
    "doesn't show a refetch error when the host $name",
    async ({ laterStatus, pollMs }) => {
      stubQueries(mockNeverFetchedWindowsHost("online"));
      (hostAPI.loadHostDetails as jest.Mock)
        .mockResolvedValueOnce({ host: mockNeverFetchedWindowsHost("online") })
        .mockResolvedValue({ host: mockNeverFetchedWindowsHost(laterStatus) });

      renderHostDetails({
        currentUser: ADMIN,
        isGlobalAdmin: true,
      });
      await screen.findByText(/fetching fresh vitals/i);
      elapsedMs = pollMs;
      // The spinner clears only once the next response has gone through the toast decision.
      await waitFor(
        () =>
          expect(
            screen.queryByText(/fetching fresh vitals/i)
          ).not.toBeInTheDocument(),
        { timeout: 5000 }
      );

      expect(notify.error).not.toHaveBeenCalled();
    },
    15000
  );
});

describe("HostDetailsPage - Show MDM commands toggle", () => {
  afterEach(() => {
    local.clear();
    jest.resetAllMocks();
  });

  it("keeps the toggle on across a remount", async () => {
    stubQueries(mockAppleHost());

    const { user, unmount } = renderHostDetails({
      isMacMdmEnabledAndConfigured: true,
    });
    const [toggle] = await screen.findAllByRole("switch");
    expect(toggle).not.toBeChecked();
    await user.click(toggle);

    expect(getMDMCommandsToggleLocalState()).toBe(true);
    expect(local.getItem("hostDetailsShowMDMCommands")).toBe("true");
    unmount();

    renderHostDetails({
      isMacMdmEnabledAndConfigured: true,
    });
    await waitFor(() => {
      expect(screen.getAllByRole("switch")[0]).toBeChecked();
    });
  });

  it("leaves the past activity feed alone on a host with no MDM commands", async () => {
    setMDMCommandsToggleLocalState(true);
    stubQueries(mockWindowsHost());

    renderHostDetails();

    expect(await screen.findByText("No activity")).toBeInTheDocument();
    expect(screen.queryAllByRole("switch")).toHaveLength(0);
  });
});

describe("HostDetailsPage - disk encryption key rotation", () => {
  afterEach(() => {
    jest.resetAllMocks();
  });

  const mockMacWithKey = (keyAvailable: boolean) => {
    const host = mockAppleHost();
    host.mdm.encryption_key_available = keyAvailable;
    host.mdm.encryption_key_archived = !keyAvailable;
    return host;
  };

  const openDiskEncryptionKeyModal = async (
    user: ReturnType<typeof userEvent.setup>
  ) => {
    await user.click(await screen.findByText("Actions"));
    await user.click(await screen.findByText("Show disk encryption key"));
    await screen.findByText("Disk encryption key");
    await waitFor(() => expect(hostAPI.getEncryptionKey).toHaveBeenCalled());
  };

  beforeEach(() => {
    (hostAPI.getEncryptionKey as jest.Mock).mockResolvedValue({
      host_id: 1,
      encryption_key: {
        key: "AAAA-BBBB-CCCC",
        updated_at: "2026-09-20T13:00:00Z",
        rotation_pending: false,
      },
    });
  });

  it("offers Rotate key to an admin when the host's key is available", async () => {
    stubQueries(mockMacWithKey(true));
    const { user } = renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });

    await openDiskEncryptionKeyModal(user);

    expect(
      await screen.findByRole("button", { name: "Rotate key" })
    ).toBeInTheDocument();
  });

  it("doesn't offer Rotate key to an observer", async () => {
    stubQueries(mockMacWithKey(true));
    const { user } = renderHostDetails({
      currentUser: OBSERVER,
      isGlobalAdmin: false,
    });

    await openDiskEncryptionKeyModal(user);

    expect(
      screen.queryByRole("button", { name: "Rotate key" })
    ).not.toBeInTheDocument();
  });

  it("doesn't offer Rotate key when only an archived key is shown", async () => {
    stubQueries(mockMacWithKey(false));
    const { user } = renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });

    await openDiskEncryptionKeyModal(user);

    expect(
      screen.queryByRole("button", { name: "Rotate key" })
    ).not.toBeInTheDocument();
  });

  it("doesn't offer Rotate key on a personal host", async () => {
    const host = mockMacWithKey(true);
    host.mdm.enrollment_status = "On (personal)";
    stubQueries(host);
    const { user } = renderHostDetails({
      currentUser: ADMIN,
      isGlobalAdmin: true,
    });

    await openDiskEncryptionKeyModal(user);

    expect(
      screen.queryByRole("button", { name: "Rotate key" })
    ).not.toBeInTheDocument();
  });
});

describe("HostDetailsPage - refetch cycle on tab switch", () => {
  const mockOnlineStuckHost = (): IHost =>
    createMockHost({
      id: 1,
      platform: "darwin",
      status: "online",
      refetch_requested: true,
    });

  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
    jest.resetAllMocks();
  });

  // #54676: each tab-focus re-entry into onSuccess used to schedule a fresh
  // setTimeout next to the one already pending, so polling sped up.
  it("does not stack polling loops when the tab regains focus mid-refetch", async () => {
    stubQueries(mockOnlineStuckHost());

    jest.useFakeTimers({ doNotFake: ["queueMicrotask"] });
    const start = 1_700_000_000_000;
    let now = start;
    jest.spyOn(Date, "now").mockImplementation(() => now);

    renderHostDetails();
    await screen.findByText("Vitals");

    // Dispatch three window-focus events (react-query's refetchOnWindowFocus
    // trigger). Each one re-enters onSuccess.
    for (let i = 0; i < 3; i += 1) {
      // eslint-disable-next-line no-await-in-loop
      await act(async () => {
        window.dispatchEvent(new Event("focus"));
        window.dispatchEvent(new Event("visibilitychange"));
        await jest.advanceTimersByTimeAsync(0);
      });
    }

    const callsBeforeAdvance = (hostAPI.loadHostDetails as jest.Mock).mock.calls
      .length;

    // One polling interval. With the fix, exactly one scheduled poll fires in
    // this window. Without it, each stacked timer fires an extra request.
    now += 2000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(2000);
    });

    const pollsInOneInterval =
      (hostAPI.loadHostDetails as jest.Mock).mock.calls.length -
      callsBeforeAdvance;
    expect(pollsInOneInterval).toBe(1);
  });

  // #54677: after give-up, the server still reports refetch_requested: true,
  // so a focus-triggered onSuccess used to re-enter the "timer just started"
  // branch and run a fresh 60s cycle (new toast 60s later).
  it("doesn't restart the refetch window after a timeout when the tab regains focus", async () => {
    jest.useFakeTimers({ doNotFake: ["queueMicrotask"] });
    const start = 1_700_000_000_000;
    let now = start;
    jest.spyOn(Date, "now").mockImplementation(() => now);

    stubQueries(mockOnlineStuckHost());
    (hostAPI.loadHostDetails as jest.Mock).mockResolvedValue({
      host: mockOnlineStuckHost(),
    });

    renderHostDetails();
    await screen.findByText("Vitals");

    // Advance past the 60s give-up window so the next polling tick fires the toast.
    now += 61000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(2000);
    });

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Refetch sent but vitals are taking longer than expected to load. You’ll see an update when the host responds."
      );
    });
    expect(notify.error).toHaveBeenCalledTimes(1);

    // Simulate a tab switch and let a full fresh 60s + 2s poll interval pass.
    // Without the fix, this re-enters the "timer just started" branch, opens a
    // new 60s cycle, and 60s later shows the toast a second time.
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
      window.dispatchEvent(new Event("visibilitychange"));
      await jest.advanceTimersByTimeAsync(0);
    });
    now += 65000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(2000);
    });

    expect(notify.error).toHaveBeenCalledTimes(1);
  });
});

describe("HostDetailsPage - enrollment rejection details", () => {
  afterEach(() => {
    jest.resetAllMocks();
  });

  it("names the Fleetd enroll secret profile for a Windows spent-secret rejection", async () => {
    stubQueries(mockWindowsHost());
    (activitiesAPI.getHostPastActivities as jest.Mock).mockResolvedValue({
      activities: [
        createMockActivity({
          type: ActivityType.HostEnrollmentRejected,
          fleet_initiated: true,
          details: {
            host_display_name: "Anna's laptop",
            reason: "one_time_secret_spent",
            platform: "windows",
          },
        }),
      ],
      meta: { has_next_results: false, has_previous_results: false },
    });

    renderHostDetails();

    await userEvent.click(
      await screen.findByRole("button", { name: "show info" })
    );

    expect(await screen.findByText("Enrollment details")).toBeInTheDocument();
    expect(screen.getByText("Fleetd enroll secret")).toBeInTheDocument();
    expect(screen.queryByText("Fleetd configuration")).not.toBeInTheDocument();
  });
});
