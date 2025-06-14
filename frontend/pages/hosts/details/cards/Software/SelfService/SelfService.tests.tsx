import React from "react";
import { screen } from "@testing-library/react";

import { noop } from "lodash";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";

import SelfService, { ISoftwareSelfServiceProps } from "./SelfService";

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
  },
  router: createMockRouter(),
  onShowInstallerDetails: noop,
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

  it("should render the contact link text if contact url is provided", () => {
    mockServer.use(customDeviceSoftwareHandler());

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} router={createMockRouter()} />);

    expect(screen.getByText("reach out to IT")).toBeInTheDocument();
    expect(screen.getByText("reach out to IT").getAttribute("href")).toBe(
      "http://example.com"
    );
  });

  it("renders installed status and 'Reinstall' and 'Uninstall' action buttons with 'installed'", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "installed",
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
        }}
        router={createMockRouter()}
        onShowInstallerDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("self-service-table__status--test")
    ).toHaveTextContent("Installed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
  });

  it("renders failed status and 'Retry' and 'Uninstall' action buttons with 'failed_install'", async () => {
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
      screen.getByTestId("self-service-table__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
  });

  it("renders failed status and 'Install' and 'Retry uninstall' action buttons with 'failed_uninstall' status", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "failed_uninstall",
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findByText("test-software");

    expect(
      screen.getByTestId("self-service-table__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Install" })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: "Retry uninstall" })
    ).toBeEnabled();
  });

  it("renders no status and 'Install' and 'Uninstall' action buttons with no API status", async () => {
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
      screen.queryByTestId("self-service-table__status--test")
    ).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Install" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
  });

  it("renders installing status and disables action buttons with 'pending_install'", async () => {
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
      screen.getByTestId("self-service-table__status--test")
    ).toHaveTextContent("Installing...");

    expect(screen.getByRole("button", { name: "Install" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeDisabled();
  });

  it("renders uninstalling status and disables 'Reinstall' and 'Uninstall' action buttons with 'pending_uninstall'", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({
            name: "test-software",
            status: "pending_uninstall",
          }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });
    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("self-service-table__status--test")
    ).toHaveTextContent("Uninstalling...");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Uninstall" })).toBeDisabled();
  });
});
