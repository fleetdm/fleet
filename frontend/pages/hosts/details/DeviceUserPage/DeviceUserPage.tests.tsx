import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IDUPDetails, IHostDevice } from "interfaces/host";
import createMockHost from "__mocks__/hostMock";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import createMockLicense from "__mocks__/licenseMock";

import { IGetSetupExperienceStatusesResponse } from "services/entities/device_user";

import { IHostPolicy } from "interfaces/policy";

import {
  customDeviceHandler,
  customDevicePastActivitiesHandler,
  customDeviceUpcomingActivitiesHandler,
  defaultDeviceCertificatesHandler,
  defaultDeviceHandler,
  defaultDevicePastActivitiesHandler,
  defaultDeviceUpcomingActivitiesHandler,
  deviceSetupExperienceHandler,
  emptySetupExperienceHandler,
} from "test/handlers/device-handler";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";
import { http, HttpResponse } from "msw";
import DeviceUserPage from "./DeviceUserPage";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";

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

// Required for tests that use useIsMobileWidth
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: jest.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
      addListener: jest.fn(), // for older APIs
      removeListener: jest.fn(),
      onchange: null,
      dispatchEvent: jest.fn(),
    })),
  });
});

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

  describe("Activity card", () => {
    const detailsLocation = {
      ...mockLocation,
      pathname: "/device/testToken",
    };

    const setupActivityTest = () => {
      mockServer.use(defaultDeviceHandler);
      mockServer.use(defaultDeviceCertificatesHandler);
      mockServer.use(emptySetupExperienceHandler);
    };

    it("renders the Activity card with Past and Upcoming tabs", async () => {
      setupActivityTest();
      mockServer.use(defaultDevicePastActivitiesHandler);
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      // The activity card header is rendered inside the Details tab panel.
      expect(await screen.findByText("Activity")).toBeInTheDocument();
      // Both sub-tabs of the activity card are present.
      expect(screen.getByText("Past")).toBeInTheDocument();
      expect(screen.getByText("Upcoming")).toBeInTheDocument();
    });

    it("renders both Activity and User cards in the Details panel", async () => {
      setupActivityTest();
      mockServer.use(defaultDevicePastActivitiesHandler);
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      expect(await screen.findByText("Activity")).toBeInTheDocument();
      expect(await screen.findByText("User")).toBeInTheDocument();
    });

    it("renders past activity items returned by the device endpoint", async () => {
      setupActivityTest();
      mockServer.use(
        customDevicePastActivitiesHandler({
          activities: [
            createMockHostPastActivity({
              id: 101,
              actor_full_name: "Admin User",
            }),
          ],
        })
      );
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      // The past activity actor should appear inside the activity feed.
      expect(await screen.findByText(/Admin User/)).toBeInTheDocument();
    });

    it("opens the software install details modal when clicking the info icon on an installed_software activity", async () => {
      setupActivityTest();
      mockServer.use(
        customDevicePastActivitiesHandler({
          activities: [
            createMockHostPastActivity({
              id: 202,
              actor_full_name: "Admin User",
              type: ActivityType.InstalledSoftware,
              details: {
                install_uuid: "test-install-uuid",
                software_title: "Test Software",
                status: "installed",
              },
            }),
          ],
        })
      );
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);
      // Stub the device install-result lookup the modal makes once opened, so
      // the test isn't tripped by an unhandled request.
      mockServer.use(
        http.get(baseUrl("/device/:token/software/install/:uuid/results"), () =>
          HttpResponse.json({ results: {} })
        )
      );

      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      // Click the "show info" button on the activity item.
      const infoButton = await screen.findByRole("button", {
        name: /show info/i,
      });
      await user.click(infoButton);

      // The modal opens with the software install details dialog.
      expect(await screen.findByText("Install details")).toBeInTheDocument();
    });

    it("hides the show-info icon for ran_script activities (no device-mode modal)", async () => {
      setupActivityTest();
      mockServer.use(
        customDevicePastActivitiesHandler({
          activities: [
            createMockHostPastActivity({
              id: 303,
              actor_full_name: "Admin User",
              type: ActivityType.RanScript,
              details: {
                script_name: "noop.sh",
                script_execution_id: "exec-noop",
              },
            }),
          ],
        })
      );
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      // The activity row renders, but the info button is hidden.
      expect(await screen.findByText(/Admin User/)).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /show info/i })
      ).not.toBeInTheDocument();
    });

    it("hides the show-info icon for failed enrollment profile renewal activities", async () => {
      setupActivityTest();
      mockServer.use(
        customDevicePastActivitiesHandler({
          activities: [
            createMockHostPastActivity({
              id: 404,
              actor_full_name: "Fleet",
              type: ActivityType.FailedEnrollmentProfileRenewal,
              details: { command_uuid: "cmd-fail" },
            }),
          ],
        })
      );
      mockServer.use(defaultDeviceUpcomingActivitiesHandler);

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      expect(
        await screen.findByText(/enrollment profile renewal failed/i)
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /show info/i })
      ).not.toBeInTheDocument();
    });

    it("disables the Upcoming tab on Android hosts", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "android";

      mockServer.use(customDeviceHandler({ host }));
      mockServer.use(defaultDeviceCertificatesHandler);
      mockServer.use(emptySetupExperienceHandler);
      mockServer.use(defaultDevicePastActivitiesHandler);
      mockServer.use(
        customDeviceUpcomingActivitiesHandler({ activities: [], count: 0 })
      );

      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={detailsLocation}
        />
      );

      await screen.findByText("Activity");
      const upcomingTab = (await screen.findByText("Upcoming")).closest(
        '[role="tab"]'
      );
      expect(upcomingTab).toHaveAttribute("aria-disabled", "true");
    });
  });
});
