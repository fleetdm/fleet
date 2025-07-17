import React from "react";
import { screen } from "@testing-library/react";

import { noop } from "lodash";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";

import SelfService, { ISoftwareSelfServiceProps } from "./SelfService";
import { DEFAULT_INSTALLED_VERSION } from "__mocks__/hostMock";

const TEST_PROPS: ISoftwareSelfServiceProps = {
  contactUrl: "http://example.com",
  deviceToken: "123-456",
  isSoftwareEnabled: true,
  pathname: "/test",
  queryParams: {
    page: 1,
    query: "",
    order_key: "name",
    order_direction: "asc",
    per_page: 10,
    vulnerable: true,
    available_for_install: false,
    min_cvss_score: undefined,
    max_cvss_score: undefined,
    exploit: false,
    category_id: undefined,
    self_service: false,
  },
  router: createMockRouter(),
  onShowInstallDetails: noop,
  onShowUninstallDetails: noop,
};

describe("SelfService", () => {
  it("should render the self service items correctly", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({ name: "test1" }),
          createMockDeviceSoftware({ name: "test2" }),
          createMockDeviceSoftware({ name: "test3" }),
        ],
        count: 3,
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findByText("test1");

    expect(true).toBe(true);
    expect(screen.getByText("test1")).toBeInTheDocument();
    expect(screen.getByText("test2")).toBeInTheDocument();
    expect(screen.getByText("test3")).toBeInTheDocument();
    screen.debug();
  });

  it("should render the contact link text for each section if contact url is provided", () => {
    mockServer.use(customDeviceSoftwareHandler());

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} router={createMockRouter()} />);

    const links = screen.getAllByRole("link", { name: /reach out to IT/i });
    expect(links.length).toBe(2);
    links.forEach((link) => {
      expect(link).toHaveAttribute("href", "http://example.com");
    });
  });

  it("renders installed status and 'Reinstall' and 'Uninstall' action buttons with 'installed' status and installed_versions", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "installed",
            installed_versions: [DEFAULT_INSTALLED_VERSION],
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });

    const expectedUrl = "http://example.com";

    render(
      <SelfService
        contactUrl={expectedUrl}
        deviceToken="123-456"
        isSoftwareEnabled
        pathname="/test"
        queryParams={{
          page: 1,
          query: "test",
          order_key: "name",
          order_direction: "asc",
          per_page: 10,
          vulnerable: true,
          available_for_install: false,
          min_cvss_score: undefined,
          max_cvss_score: undefined,
          exploit: false,
          category_id: undefined,
          self_service: false,
        }}
        router={createMockRouter()}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Installed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
  });

  it("renders installed status and 'Reinstall' and 'Uninstall' action buttons with null status and installed_versions", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: null,
            installed_versions: [DEFAULT_INSTALLED_VERSION],
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });

    const expectedUrl = "http://example.com";

    render(
      <SelfService
        contactUrl={expectedUrl}
        deviceToken="123-456"
        isSoftwareEnabled
        pathname="/test"
        queryParams={{
          page: 1,
          query: "test",
          order_key: "name",
          order_direction: "asc",
          per_page: 10,
          vulnerable: true,
          available_for_install: false,
          min_cvss_score: undefined,
          max_cvss_score: undefined,
          exploit: false,
          category_id: undefined,
          self_service: false,
        }}
        router={createMockRouter()}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Installed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
  });

  it("renders failed status, 'Retry' button and hides 'Uninstall' button with 'failed_install' and no installed versions detected", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "failed_install",
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
    expect(
      screen.queryByRole("button", { name: "Uninstall" })
    ).not.toBeInTheDocument();
  });

  it("renders failed status and 'Install' and 'Retry uninstall' action buttons with 'failed_uninstall' status and installed_versions detected", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "failed_uninstall",
            installed_versions: [DEFAULT_INSTALLED_VERSION],
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: "Retry uninstall" })
    ).toBeEnabled();
  });

  it("renders no status, 'Install' action, and no 'Uninstall' action with no API status and no installed_versions", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: null,
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.queryByTestId("install-status-cell__status--test")
    ).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Install" })).toBeEnabled();
    expect(
      screen.queryByRole("button", { name: "Uninstall" })
    ).not.toBeInTheDocument();
  });

  it("renders installing status, disables Install action, and hides Uninstall action with 'pending_install' and no installed_version", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "pending_install",
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Installing...");

    expect(screen.getByRole("button", { name: "Install" })).toBeDisabled();
    expect(
      screen.queryByRole("button", { name: "Uninstall" })
    ).not.toBeInTheDocument();
  });

  it("renders uninstalling status and disables 'Reinstall' and 'Uninstall' action buttons with 'pending_uninstall'", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "pending_uninstall",
            installed_versions: [DEFAULT_INSTALLED_VERSION], // Uninstall requires installed versions
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Uninstalling...");

    expect(screen.getByRole("button", { name: "Install" })).toBeDisabled(); // TODO: Should this say "Reinstall"?
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeDisabled();
  });
});
