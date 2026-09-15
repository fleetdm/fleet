import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockHost from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
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

const mockAndroidHost = (): IHost => {
  const host = createMockHost({ platform: "android", status: "online" });
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
  isAndroidMdmEnabledAndConfigured?: boolean;
}) => {
  const {
    currentUser = ADMIN,
    isGlobalAdmin = true,
    isMacMdmEnabledAndConfigured = false,
    isAndroidMdmEnabledAndConfigured = false,
  } = overrides || {};

  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser,
        isGlobalAdmin,
        isPremiumTier: true,
        isMacMdmEnabledAndConfigured,
        isAndroidMdmEnabledAndConfigured,
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

describe("HostDetailsPage - Android MDM commands", () => {
  // the exact product copy, deliberately duplicated so a change to it is a
  // conscious one
  const ANDROID_TOOLTIP =
    "Activities and non-custom MDM commands are not supported yet for Android.";

  const renderAndroidHostDetails = () =>
    renderHostDetails({ isAndroidMdmEnabledAndConfigured: true });

  const emptyCommands = {
    results: [],
    count: 0,
    meta: { has_next_results: false, has_previous_results: false },
  };

  /** Only the "pending" filter returns a command, so the Upcoming feed is the
   * only place its row can come from. */
  const stubPendingAndroidCommand = () => {
    (commandAPI.getCommands as jest.Mock).mockImplementation(
      ({ command_status }: { command_status: string }) =>
        Promise.resolve(
          command_status === "pending"
            ? {
                results: [
                  {
                    host_uuid: mockAndroidHost().uuid,
                    command_uuid: "android-command-uuid",
                    command_status: "pending",
                    status: "Pending",
                    updated_at: "2024-01-01T00:00:00Z",
                    request_type: "LOCK",
                    hostname: "android-host",
                    name: null,
                  },
                ],
                count: 1,
                meta: { has_next_results: false, has_previous_results: false },
              }
            : emptyCommands
        )
    );
  };

  afterEach(() => {
    local.clear();
    jest.resetAllMocks();
  });

  it("shows the toggle pinned on and disabled", async () => {
    // even with the stored preference off, Android has nothing else to show
    setMDMCommandsToggleLocalState(false);
    stubQueries(mockAndroidHost());

    const { user } = renderAndroidHostDetails();

    const [toggle] = await screen.findAllByRole("switch");
    expect(toggle).toBeChecked();
    expect(toggle).toBeDisabled();

    await user.hover(screen.getAllByText(/Show MDM commands/)[0]);
    expect(await screen.findByText(ANDROID_TOOLTIP)).toBeInTheDocument();
  });

  it("pins the toggle on without writing to the stored preference", async () => {
    setMDMCommandsToggleLocalState(false);
    stubQueries(mockAndroidHost());

    const { unmount } = renderAndroidHostDetails();

    const [toggle] = await screen.findAllByRole("switch");
    expect(toggle).toBeChecked();
    expect(getMDMCommandsToggleLocalState()).toBe(false);
    expect(local.getItem("hostDetailsShowMDMCommands")).toBe("false");
    unmount();

    // an Apple host viewed afterwards still honors the untouched preference
    stubQueries(mockAppleHost());
    renderHostDetails({ isMacMdmEnabledAndConfigured: true });
    await waitFor(() => {
      expect(screen.getAllByRole("switch")[0]).not.toBeChecked();
    });
    expect(screen.getAllByRole("switch")[0]).toBeEnabled();
  });

  it("requests ran and failed commands for the past tab", async () => {
    stubQueries(mockAndroidHost());

    renderAndroidHostDetails();

    await waitFor(() => {
      expect(commandAPI.getCommands).toHaveBeenCalledWith(
        expect.objectContaining({
          host_identifier: mockAndroidHost().uuid,
          command_status: "ran,failed",
        })
      );
    });
  });

  it("lists pending commands under an enabled upcoming tab", async () => {
    stubQueries(mockAndroidHost());
    stubPendingAndroidCommand();

    const { user } = renderAndroidHostDetails();

    const upcomingTab = await screen.findByRole("tab", { name: /upcoming/i });
    expect(upcomingTab).toHaveAttribute("aria-disabled", "false");
    // the Past tab asks for ran,failed, which returns nothing here
    expect(await screen.findByText("No MDM commands")).toBeInTheDocument();
    expect(screen.queryByText("LOCK")).not.toBeInTheDocument();

    await user.click(upcomingTab);

    await waitFor(() => {
      expect(commandAPI.getCommands).toHaveBeenCalledWith(
        expect.objectContaining({
          host_identifier: mockAndroidHost().uuid,
          command_status: "pending",
        })
      );
    });
    expect(await screen.findByText("LOCK")).toBeInTheDocument();
  });

  it("hides the toggle and the command feed when Android MDM is not configured", async () => {
    // a stale response from a previously-viewed host must not leak through
    setMDMCommandsToggleLocalState(true);
    stubQueries(mockAndroidHost());

    renderHostDetails({ isAndroidMdmEnabledAndConfigured: false });

    expect(await screen.findByText("No activity")).toBeInTheDocument();
    expect(screen.queryAllByRole("switch")).toHaveLength(0);
    expect(commandAPI.getCommands).not.toHaveBeenCalled();
  });
});
