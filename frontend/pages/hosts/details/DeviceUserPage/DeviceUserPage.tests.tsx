import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IDeviceUserResponse, IHostDevice } from "interfaces/host";
import createMockHost from "__mocks__/hostMock";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockLicense from "__mocks__/licenseMock";

import { IGetSetupExperienceStatusesResponse } from "services/entities/device_user";

import {
  customDeviceHandler,
  defaultDeviceCertificatesHandler,
  defaultDeviceHandler,
  deviceSetupExperienceHandler,
  emptySetupExperienceHandler,
} from "test/handlers/device-handler";
import DeviceUserPage from "./DeviceUserPage";

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
    const REGULAR_DUP_MATCHER = /Last fetched/;
    const SETTING_UP_YOUR_DEVICE_MATCHER = /Setting up your device/;
    const CONFIG_COMPLETE_MATCHER = /Configuration complete/;
    const SETUP_FAILED_MATCHER = /Device setup failed/;

    const setupTest = async (
      deviceUserResponseOverrides?: Partial<IDeviceUserResponse>,
      setupExperienceOverrides?: Partial<IGetSetupExperienceStatusesResponse>,
      mockLocationOverrides = {}
    ) => {
      mockServer.use(customDeviceHandler(deviceUserResponseOverrides));
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
        expect(screen.getByText(/Installing/)).toBeInTheDocument();
        expect(screen.getByText(/Running/)).toBeInTheDocument();
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
              { type: "software_installer", name: "step 1", status: "success" },
            ],
            scripts: [{ type: "script", name: "step 2", status: "failure" }],
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
    it("renders the setup experience failure page if some software failed and require_all_software_macos is true and the host is a mac", async () => {
      const host = createMockHost() as IHostDevice;
      host.platform = "darwin";

      await setupTest(
        {
          host,
          global_config: {
            features: { enable_software_inventory: true },
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
    const setupTest = async (overrides: Partial<IDeviceUserResponse>) => {
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
          features: { enable_software_inventory: true },
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
          features: { enable_software_inventory: true },
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
          features: { enable_software_inventory: true },
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
          features: { enable_software_inventory: true },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });
  });
});
