import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IDUPDetails, IHostDevice } from "interfaces/host";
import createMockHost from "__mocks__/hostMock";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockLicense from "__mocks__/licenseMock";
import { notify } from "components/ToastNotification";
import { HostPlatform } from "interfaces/platform";

import deviceUserAPI, {
  IGetSetupExperienceStatusesResponse,
} from "services/entities/device_user";

import { IHostPolicy } from "interfaces/policy";

import {
  customDeviceHandler,
  defaultDeviceCertificatesHandler,
  defaultDeviceHandler,
  deviceSetupExperienceHandler,
  emptySetupExperienceHandler,
  ssoRequiredDeviceCertificatesHandler,
  ssoRequiredDeviceHandler,
  unauthorizedDeviceHandler,
} from "test/handlers/device-handler";
import DeviceUserPage from "./DeviceUserPage";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const mockRouter = createMockRouter();

const mockLocation = {
  pathname: "",
  query: {
    vulnerable: undefined,
    page: undefined,
    query: undefined,
    order_key: undefined,
    order_direction: undefined,
    setup_only: "",
  },
  search: undefined,
};

describe("Device User Page", () => {
  it("hides the software tab if the device has no software", async () => {
    mockServer.use(defaultDeviceHandler);
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );

    // waiting for the device data to render
    await screen.findByText(/Details/);

    expect(screen.queryByText(/Software/)).not.toBeInTheDocument();
  });

  it("hides the certificates card if the device has no certificates", async () => {
    mockServer.use(defaultDeviceHandler);
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );

    // waiting for the device data to render
    await screen.findByText(/Details/);

    expect(screen.queryByText(/Certificates/)).not.toBeInTheDocument();
  });

  it("hides the certificates card if the device is not an apple device (mac, iphone, ipad)", async () => {
    const host = createMockHost() as IHostDevice;
    host.mdm.enrollment_status = "On (manual)";
    host.platform = "windows";
    host.dep_assigned_to_fleet = false;

    mockServer.use(customDeviceHandler({ host }));
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );

    // waiting for the device data to render
    await screen.findByText(/Details/);

    expect(screen.queryByText(/Certificates/)).not.toBeInTheDocument();
  });

  it("hides the user card if the device is not apple or android device", async () => {
    const host = createMockHost() as IHostDevice;
    host.platform = "windows";
    host.end_users = [];

    mockServer.use(customDeviceHandler({ host }));
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );

    // waiting for the device data to render
    await screen.findByText(/Details/);

    expect(screen.queryByText(/User/)).not.toBeInTheDocument();
  });

  describe("Setup experience software installation", () => {
    const REGULAR_DUP_MATCHER = /Last fetched/i;
    const SETTING_UP_YOUR_DEVICE_MATCHER = /Setting up your device/i;
    const CONFIG_COMPLETE_MATCHER = /Configuration complete/i;
    const SETUP_FAILED_MATCHER = /Device setup failed/i;

    const setupTest = async (
      dupDetailsOverrides?: Partial<IDUPDetails>,
      setupExperienceOverrides?: Partial<IGetSetupExperienceStatusesResponse>,
      mockLocationOverrides = {}
    ) => {
      mockServer.use(customDeviceHandler(dupDetailsOverrides));
      mockServer.use(defaultDeviceCertificatesHandler);
      mockServer.use(deviceSetupExperienceHandler(setupExperienceOverrides));

      const render = createCustomRenderer({
        withBackendMock: true,
      });

      const { user } = render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={{
            ...(mockLocation || {}),
            ...(mockLocationOverrides || {}),
          }}
        />
      );

      return user;
    };

    it("does not check for setup experience software on Fleet Free", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest({ host, license: createMockLicense({ tier: "free" }) });

      await waitFor(() => {
        expect(screen.getByText(REGULAR_DUP_MATCHER)).toBeInTheDocument();
      });
    });

    it("checks for setup experience steps on Fleet Premium, and renders Setting Up Your Device if there are such steps", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest({ host });

      await waitFor(() => {
        expect(
          screen.getByText(SETTING_UP_YOUR_DEVICE_MATCHER)
        ).toBeInTheDocument();
        expect(screen.getAllByText(/Install/i).length).toBeGreaterThan(0);
        expect(screen.getAllByText(/Run/i).length).toBeGreaterThan(0);
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
    it("checks for setup experience steps on Fleet Premium, and renders the normal device user page if there are such steps", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest(
        { host },
        { setup_experience_results: { software: [], scripts: [] } }
      );

      await waitFor(() => {
        expect(screen.getByText(REGULAR_DUP_MATCHER)).toBeInTheDocument();
      });

      expect(screen.queryByText(SETTING_UP_YOUR_DEVICE_MATCHER)).toBeNull();
    });
    it("checks for setup experience steps on Fleet Premium, and renders Setting Up Your Device even if there are no such steps if setup_only=1 is in the query", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest(
        { host },
        { setup_experience_results: { software: [], scripts: [] } },
        { query: { setup_only: "1" } }
      );
      await waitFor(() => {
        expect(screen.getByText(CONFIG_COMPLETE_MATCHER)).toBeInTheDocument();
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
    it("checks for setup experience items on Fleet Premium, and renders Setting Up Your Device when all steps are complete if setup_only=1 is in the query", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest(
        { host },
        {
          setup_experience_results: {
            software: [
              { name: "step 1.sh", status: "success", type: "script_run" },
            ],
            scripts: [
              { name: "step 2.sh", status: "failure", type: "script_run" },
            ],
          },
        },
        { query: { setup_only: "1" } }
      );
      await waitFor(() => {
        expect(screen.getByText(CONFIG_COMPLETE_MATCHER)).toBeInTheDocument();
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
    it("renders the regular setup experience page if some software failed and require_all_software_macos is not true", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "darwin";

      await setupTest(
        { host },
        {
          setup_experience_results: {
            software: [
              {
                type: "software_installer",
                name: "step 1",
                status: "success",
              },
              {
                type: "software_installer",
                name: "step 2",
                status: "failure",
                error: "error message",
              },
              {
                type: "software_installer",
                name: "step 3",
                status: "pending",
              },
            ],
            scripts: [],
          },
        }
      );
      await waitFor(() => {
        expect(
          screen.getByText(SETTING_UP_YOUR_DEVICE_MATCHER)
        ).toBeInTheDocument();
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
    it("renders the regular setup experience page if some software failed and require_all_software_macos is true but the device is not on macos", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "linux";

      await setupTest(
        {
          host,
          global_config: {
            features: {
              enable_software_inventory: true,
              enable_conditional_access: false,
              enable_conditional_access_bypass: false,
            },
            mdm: {
              enabled_and_configured: true,
              require_all_software_macos: true,
            },
          },
        },
        {
          setup_experience_results: {
            software: [
              {
                type: "software_installer",
                name: "step 1",
                status: "success",
              },
              {
                type: "software_installer",
                name: "step 2",
                status: "failure",
                error: "error message",
              },
              {
                type: "software_installer",
                name: "step 3",
                status: "pending",
              },
            ],
            scripts: [],
          },
        }
      );
      await waitFor(() => {
        expect(
          screen.getByText(SETTING_UP_YOUR_DEVICE_MATCHER)
        ).toBeInTheDocument();
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
    it("renders the setup experience failure page if some software failed and require_all_software_macos is true and the host is a mac", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "darwin";

      await setupTest(
        {
          host,
          global_config: {
            features: {
              enable_software_inventory: true,
              enable_conditional_access: false,
              enable_conditional_access_bypass: false,
            },
            mdm: {
              enabled_and_configured: true,
              require_all_software_macos: true,
            },
          },
        },
        {
          setup_experience_results: {
            software: [
              {
                type: "software_installer",
                name: "step 1",
                status: "success",
              },
              {
                type: "software_installer",
                name: "step 2",
                status: "failure",
                error: "error message",
              },
              {
                type: "software_installer",
                name: "step 3",
                status: "pending",
              },
            ],
            scripts: [],
          },
        }
      );
      await waitFor(() => {
        expect(screen.getByText(SETUP_FAILED_MATCHER)).toBeInTheDocument();
        const detailsButton = screen.getByRole("button", { name: /details/i });
        expect(detailsButton).toBeInTheDocument();
        // CLick the details button to show the error message
        detailsButton.click();
        expect(screen.getByText(/error message/i)).toBeInTheDocument();
      });

      expect(screen.queryByText(REGULAR_DUP_MATCHER)).toBeNull();
    });
  });

  describe("MDM enrollment", () => {
    const setupTest = async (overrides: Partial<IDUPDetails>) => {
      mockServer.use(customDeviceHandler(overrides));
      mockServer.use(defaultDeviceCertificatesHandler);
      mockServer.use(emptySetupExperienceHandler);

      const render = createCustomRenderer({
        withBackendMock: true,
      });

      const { user } = await render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={mockLocation}
        />
      );

      // waiting for the device data to render
      await screen.findByText(/Details/);

      return user;
    };

    it("shows a banner when MDM is configured and the device is unenrolled", async () => {
      const host = createMockHost() as IHostDevice;
      host.mdm.enrollment_status = "Off";
      host.platform = "darwin";
      host.dep_assigned_to_fleet = false;

      const user = await setupTest({
        host,
        global_config: {
          mdm: {
            enabled_and_configured: true,
            require_all_software_macos: false,
          },
          features: {
            enable_software_inventory: true,
            enable_conditional_access: false,
            enable_conditional_access_bypass: false,
          },
        },
      });

      await user.click(screen.getByRole("button", { name: "Turn on MDM" }));
    });

    it("shows a banner when MDM is configured and the device doesn't have MDM info", async () => {
      const host = createMockHost() as IHostDevice;
      host.mdm.enrollment_status = null;
      host.platform = "darwin";
      host.dep_assigned_to_fleet = false;

      const user = await setupTest({
        host,
        global_config: {
          mdm: {
            enabled_and_configured: true,
            require_all_software_macos: false,
          },
          features: {
            enable_software_inventory: true,
            enable_conditional_access: false,
            enable_conditional_access_bypass: false,
          },
        },
      });

      await user.click(screen.getByRole("button", { name: "Turn on MDM" }));
    });

    it("doesn't  show a banner when MDM is not configured", async () => {
      const host = createMockHost() as IHostDevice;
      host.mdm.enrollment_status = null;
      host.platform = "darwin";
      host.dep_assigned_to_fleet = false;

      await setupTest({
        host,
        global_config: {
          mdm: {
            enabled_and_configured: false,
            require_all_software_macos: false,
          },
          features: {
            enable_software_inventory: true,
            enable_conditional_access: false,
            enable_conditional_access_bypass: false,
          },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });

    it("doesn't  show a banner when the host already has MDM enabled", async () => {
      const host = createMockHost() as IHostDevice;
      host.mdm.enrollment_status = "On (manual)";
      host.platform = "darwin";
      host.dep_assigned_to_fleet = false;

      await setupTest({
        host,
        global_config: {
          mdm: {
            enabled_and_configured: true,
            require_all_software_macos: false,
          },
          features: {
            enable_software_inventory: true,
            enable_conditional_access: false,
            enable_conditional_access_bypass: false,
          },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });
  });

  describe("Conditional access feature flags", () => {
    // Test PolicyDetailsModal directly to verify the onResolveLater behavior
    // which is controlled by enable_conditional_access and enable_conditional_access_bypass flags
    const createFailingConditionalAccessPolicy = (): IHostPolicy => ({
      id: 1,
      name: "Test Policy",
      query: "SELECT 1",
      description: "Test description",
      author_id: 1,
      author_name: "Test Author",
      author_email: "test@example.com",
      resolution: "Fix the issue",
      platform: "darwin",
      team_id: null,
      created_at: "2022-01-01T12:00:00Z",
      updated_at: "2022-01-02T12:00:00Z",
      critical: false,
      calendar_events_enabled: false,
      conditional_access_enabled: true,
      type: "dynamic",
      response: "fail",
    });

    it("shows 'Resolve later' button when onResolveLater is provided and policy is failing conditional access", () => {
      createCustomRenderer({})(
        <PolicyDetailsModal
          onCancel={jest.fn()}
          policy={createFailingConditionalAccessPolicy()}
          onResolveLater={jest.fn()}
        />
      );

      expect(
        screen.getByRole("button", { name: "Resolve later" })
      ).toBeInTheDocument();
    });

    it("does not show 'Resolve later' button when onResolveLater is not provided", () => {
      createCustomRenderer({})(
        <PolicyDetailsModal
          onCancel={jest.fn()}
          policy={createFailingConditionalAccessPolicy()}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resolve later" })
      ).not.toBeInTheDocument();
    });

    it("does not show 'Resolve later' button when policy is passing", () => {
      const passingPolicy = {
        ...createFailingConditionalAccessPolicy(),
        response: "pass" as const,
      };

      createCustomRenderer({})(
        <PolicyDetailsModal
          onCancel={jest.fn()}
          policy={passingPolicy}
          onResolveLater={jest.fn()}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resolve later" })
      ).not.toBeInTheDocument();
    });

    it("does not show 'Resolve later' button when policy does not have conditional_access_enabled", () => {
      const nonConditionalPolicy = {
        ...createFailingConditionalAccessPolicy(),
        conditional_access_enabled: false,
      };

      createCustomRenderer({})(
        <PolicyDetailsModal
          onCancel={jest.fn()}
          policy={nonConditionalPolicy}
          onResolveLater={jest.fn()}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resolve later" })
      ).not.toBeInTheDocument();
    });
  });

  describe("Vitals refetch timeout", () => {
    const REAL_NOW = new Date("2026-01-01T00:00:00Z").getTime();
    let mockNow = REAL_NOW;
    let dateNowSpy: jest.SpyInstance;

    beforeEach(() => {
      mockNow = REAL_NOW;
      dateNowSpy = jest.spyOn(Date, "now").mockImplementation(() => mockNow);
    });

    afterEach(() => {
      dateNowSpy.mockRestore();
    });

    it("shows an uncertain 'taking longer than expected' message instead of claiming failure once the poll window is exceeded", async () => {
      const host = createMockHost({
        refetch_requested: true,
        status: "online",
        platform: "ubuntu",
      }) as IHostDevice;

      mockServer.use(customDeviceHandler({ host }));
      mockServer.use(defaultDeviceCertificatesHandler);
      mockServer.use(emptySetupExperienceHandler);

      const render = createCustomRenderer({
        withBackendMock: true,
      });

      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={mockLocation}
        />
      );

      // Wait for the first successful load, which starts the refetch
      // timer and schedules the next poll via a real setTimeout.
      await screen.findByText(/Details/);

      // Jump the clock past the 3-minute give-up window before that
      // scheduled poll fires and re-evaluates elapsed time.
      mockNow += 200000;

      await waitFor(
        () => {
          expect(notify.error).toHaveBeenCalledWith(
            "Refetch sent but vitals are taking longer than expected to load. You’ll see an update when the host responds."
          );
        },
        { timeout: 4000 }
      );
    }, 10000);
  });
});

describe("Device User Page - Fleet Desktop SSO", () => {
  let initiateDeviceSSO: jest.SpyInstance;

  const pendingInitiation = () => new Promise<{ url: string }>(() => undefined);

  const renderPage = (
    query: Record<string, string> = {},
    token = "testToken"
  ) => {
    const render = createCustomRenderer({ withBackendMock: true });
    return render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: token }}
        location={{
          ...mockLocation,
          query: { ...mockLocation.query, ...query },
        }}
      />
    );
  };

  const expectRedirectingToIdP = async () =>
    expect(
      await screen.findByText(/Redirecting to your organization/)
    ).toBeInTheDocument();

  const expectManualRetry = async () =>
    expect(
      await screen.findByRole("button", { name: "Sign in again" })
    ).toBeInTheDocument();

  beforeEach(() => {
    window.sessionStorage.clear();
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);
    initiateDeviceSSO = jest
      .spyOn(deviceUserAPI, "initiateDeviceSSO")
      .mockImplementation(pendingInitiation);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("initiates SSO when a device call reports sso_required", async () => {
    mockServer.use(ssoRequiredDeviceHandler);

    renderPage();

    await expectRedirectingToIdP();
    expect(initiateDeviceSSO).toHaveBeenCalledWith("testToken");
  });

  it("initiates when a query other than the details call is the one refused", async () => {
    mockServer.use(defaultDeviceHandler);
    mockServer.use(ssoRequiredDeviceCertificatesHandler);

    renderPage();

    await expectRedirectingToIdP();
    expect(initiateDeviceSSO).toHaveBeenCalledWith("testToken");
  });

  it("renders the invalid URL error, and starts no SSO flow, for a 401 with no marker", async () => {
    mockServer.use(unauthorizedDeviceHandler);

    renderPage();

    expect(
      await screen.findByText("This URL is invalid or expired.")
    ).toBeInTheDocument();
    expect(initiateDeviceSSO).not.toHaveBeenCalled();
  });

  it("does not initiate a second time after a round-trip that left no session", async () => {
    mockServer.use(ssoRequiredDeviceHandler);
    window.sessionStorage.setItem("fleet-device-sso-attempt:testToken", "1");

    renderPage();

    await expectManualRetry();
    expect(initiateDeviceSSO).not.toHaveBeenCalled();
  });

  it("does not auto-initiate when the attempt cannot be remembered", async () => {
    mockServer.use(ssoRequiredDeviceHandler);
    jest.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    jest.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });

    const { user } = renderPage();

    const retry = await screen.findByRole("button", { name: "Sign in again" });
    expect(initiateDeviceSSO).not.toHaveBeenCalled();

    await user.click(retry);

    await expectRedirectingToIdP();
    expect(initiateDeviceSSO).toHaveBeenCalledWith("testToken");
  });

  it("does not auto-initiate when the callback reported a failure", async () => {
    mockServer.use(ssoRequiredDeviceHandler);

    renderPage({ sso_error: "server_error" });

    await expectManualRetry();
    expect(initiateDeviceSSO).not.toHaveBeenCalled();
  });

  it("does not auto-initiate during Setup Experience", async () => {
    mockServer.use(ssoRequiredDeviceHandler);

    renderPage({ setup_only: "1" });

    await expectManualRetry();
    expect(initiateDeviceSSO).not.toHaveBeenCalled();
  });

  it("renders a retryable error when the initiate call fails", async () => {
    mockServer.use(ssoRequiredDeviceHandler);
    initiateDeviceSSO.mockRejectedValueOnce(new Error("nope"));

    const { user } = renderPage();

    const retry = await screen.findByRole("button", { name: "Sign in again" });
    expect(initiateDeviceSSO).toHaveBeenCalledTimes(1);

    await user.click(retry);

    await expectRedirectingToIdP();
    expect(initiateDeviceSSO).toHaveBeenCalledTimes(2);
  });
});

describe("Device User Page - MDM check-in ping", () => {
  let apnsPingSpy: jest.SpyInstance;
  let refetchSpy: jest.SpyInstance;

  beforeEach(() => {
    apnsPingSpy = jest.spyOn(deviceUserAPI, "apnsPing").mockResolvedValue({});
    refetchSpy = jest.spyOn(deviceUserAPI, "refetch").mockResolvedValue({});
    mockServer.use(defaultDeviceCertificatesHandler);
    mockServer.use(emptySetupExperienceHandler);
  });

  afterEach(() => {
    apnsPingSpy.mockRestore();
    refetchSpy.mockRestore();
  });

  const renderDevicePage = (
    platform: HostPlatform,
    enrollmentStatus: IHostDevice["mdm"]["enrollment_status"]
  ) => {
    const host = createMockHost({
      platform,
      status: "online",
    }) as IHostDevice;
    host.mdm.enrollment_status = enrollmentStatus;
    host.mdm.connected_to_fleet = true;

    mockServer.use(customDeviceHandler({ host }));

    const render = createCustomRenderer({ withBackendMock: true });

    return render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );
  };

  it("pings APNS alongside the refetch for an MDM-enrolled Apple host", async () => {
    const { user } = renderDevicePage("darwin", "On (manual)");

    await user.click(await screen.findByRole("button", { name: /refetch/i }));

    await waitFor(() => {
      expect(apnsPingSpy).toHaveBeenCalledWith("testToken");
    });
    expect(refetchSpy).toHaveBeenCalledWith("testToken");
  });

  it("does not ping APNS when refetching a non-Apple host", async () => {
    const { user } = renderDevicePage("ubuntu", "On (manual)");

    await user.click(await screen.findByRole("button", { name: /refetch/i }));

    // The ping is fired before the refetch, so a completed refetch means the
    // ping decision has already been made.
    await waitFor(() => {
      expect(refetchSpy).toHaveBeenCalledWith("testToken");
    });
    expect(apnsPingSpy).not.toHaveBeenCalled();
  });

  it("does not ping APNS when the host's MDM enrollment is off", async () => {
    const { user } = renderDevicePage("darwin", "Off");

    await user.click(await screen.findByRole("button", { name: /refetch/i }));

    await waitFor(() => {
      expect(refetchSpy).toHaveBeenCalledWith("testToken");
    });
    expect(apnsPingSpy).not.toHaveBeenCalled();
  });

  it("does not ping APNS when the host's MDM enrollment is still pending", async () => {
    const { user } = renderDevicePage("darwin", "Pending");

    await user.click(await screen.findByRole("button", { name: /refetch/i }));

    await waitFor(() => {
      expect(refetchSpy).toHaveBeenCalledWith("testToken");
    });
    expect(apnsPingSpy).not.toHaveBeenCalled();
  });

  it("surfaces an error but still refetches when the APNS ping fails", async () => {
    apnsPingSpy.mockRejectedValue(new Error("ping failed"));

    const { user } = renderDevicePage("darwin", "On (manual)");

    await user.click(await screen.findByRole("button", { name: /refetch/i }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Failed to send APNS ping",
        expect.anything()
      );
    });
    expect(refetchSpy).toHaveBeenCalledWith("testToken");
  });
});
