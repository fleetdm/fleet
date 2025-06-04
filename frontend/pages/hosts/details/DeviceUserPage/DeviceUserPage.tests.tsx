import React from "react";
import { screen } from "@testing-library/react";

import { IDeviceUserResponse, IHostDevice } from "interfaces/host";
import createMockHost from "__mocks__/hostMock";
import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";
import {
  customDeviceHandler,
  defaultDeviceCertificatesHandler,
  defaultDeviceHandler,
} from "test/handlers/device-handler";
import DeviceUserPage from "./DeviceUserPage";

const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

const mockLocation = {
  pathname: "",
  query: {
    vulnerable: undefined,
    page: undefined,
    query: undefined,
    order_key: undefined,
    order_direction: undefined,
  },
  search: undefined,
};

describe("Device User Page", () => {
  it("hides the software tab if the device has no software", async () => {
    mockServer.use(defaultDeviceHandler);
    mockServer.use(defaultDeviceCertificatesHandler);

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

  describe("MDM enrollment", () => {
    const setupTest = async (overrides: Partial<IDeviceUserResponse>) => {
      mockServer.use(customDeviceHandler(overrides));
      mockServer.use(defaultDeviceCertificatesHandler);

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
          mdm: { enabled_and_configured: true },
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
          mdm: { enabled_and_configured: true },
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
          mdm: { enabled_and_configured: false },
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
          mdm: { enabled_and_configured: true },
          features: { enable_software_inventory: true },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });
  });
  // // FIXME: revisit these tests when we have a better way to test modals
  // describe("AutoEnrollMDMModal", () => {
  //   it("shows the pre-Sonoma body when the host is pre-Sonoma", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 13.1.1";
  //     host.dep_assigned_to_fleet = true;

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: true },
  //           features: { enable_software_inventory: false },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // autoenroll-specific copy
  //     expect(
  //       screen.getByText("sudo profiles renew -type enrollment")
  //     ).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("notification center")).toBeInTheDocument();
  //   });

  //   it("shows the Sonoma-and-above body when the host is Sonoma", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 14.7";
  //     host.dep_assigned_to_fleet = true;

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: true },
  //           features: { enable_software_inventory: false },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // autoenroll-specific copy
  //     expect(
  //       screen.getByText("sudo profiles renew -type enrollment")
  //     ).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("System Settings")).toBeInTheDocument();
  //   });

  //   it("shows the Sonoma-and-above body when the host is post-Sonoma", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 15.3";
  //     host.dep_assigned_to_fleet = true;

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: true },
  //           features: { enable_software_inventory: false },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // autoenroll-specific copy
  //     expect(
  //       screen.getByText("sudo profiles renew -type enrollment")
  //     ).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("System Settings")).toBeInTheDocument();
  //   });
  // });
  // // FIXME: revisit these tests when we have a better way to test modals
  // describe("ManualEnrollMDMModal", () => {
  //   it("shows the pre-Seqouia body when the host is pre-Seqouia", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 14.1.1";

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: false },
  //           features: { enable_software_inventory: true },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // manualenroll-specific copy
  //     expect(screen.getByText("Download your profile.")).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("In the search bar")).toBeInTheDocument();
  //   });

  //   it("shows the Sequoia-and-above body when the host is Sequoia", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 15.3";

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: false },
  //           features: { enable_software_inventory: true },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // manualenroll-specific copy
  //     expect(screen.getByText("Download your profile.")).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("In the sidebar menu")).toBeInTheDocument();
  //   });

  //   it("shows the Sequoia-and-above body when the host is post-Sequoia", async () => {
  //     const host = createMockHost() as IHostDevice;
  //     host.platform = "darwin";
  //     host.os_version = "macOS 16.0";

  //     mockServer.use(
  //       customDeviceHandler({
  //         host,
  //         global_config: {
  //           mdm: { enabled_and_configured: false },
  //           features: { enable_software_inventory: true },
  //         },
  //       })
  //     );

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //     });

  //     const { user } = render(
  //       <DeviceUserPage
  //         router={mockRouter}
  //         params={{ device_auth_token: "testToken" }}
  //         location={mockLocation}
  //       />
  //     );

  //     // waiting for the device data to render
  //     await screen.findByText("About");

  //     // open the modal
  //     await user.click(screen.getByRole("button", { name: "Turn on MDM" }));

  //     // waiting for the modal to render
  //     await screen.findByText("To turn on MDM,");

  //     // manual-specific copy
  //     expect(screen.getByText("Download your profile.")).toBeInTheDocument();
  //     // version-specific copy
  //     expect(screen.getByText("In the sidebar menu")).toBeInTheDocument();
  //   });
  // });
});
