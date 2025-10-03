import React from "react";
import { screen, within } from "@testing-library/react";

import { noop } from "lodash";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import {
  DEFAULT_INSTALLED_VERSION,
  DEFAULT_HOST_HOSTNAME,
} from "__mocks__/hostMock";

import SelfService, { ISoftwareSelfServiceProps } from "./SelfService";

/**
 * Finds the "More" actions dropdown combobox.
 * Returns the combobox or throws an error if not found.
 */
const getMoreDropdown = () => {
  const combos = screen.getAllByRole("combobox");
  const moreDropdown = combos.find((combo) => {
    const parentText = combo.parentElement && combo.parentElement.textContent;
    return !!parentText && /more/i.test(parentText);
  });
  if (!moreDropdown) {
    throw new Error("Could not find the More actions dropdown");
  }
  return moreDropdown;
};

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
    category_id: undefined,
  },
  router: createMockRouter(),
  refetchHostDetails: noop,
  isHostDetailsPolling: false,
  hostDisplayName: DEFAULT_HOST_HOSTNAME,
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
    await screen.findAllByText("test1");

    // Truncated tooltip causes multiple text rendering
    expect(screen.getAllByText("test1")).toHaveLength(2);
    expect(screen.getAllByText("test2")).toHaveLength(2);
    expect(screen.getAllByText("test3")).toHaveLength(2);
  });

  it("renders installed status and 'Reinstall' action button and 'More' dropdown with 'installed' status and installed_versions", async () => {
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

    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Installed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    const moreDropdown = getMoreDropdown();
    expect(moreDropdown).toBeEnabled();
  });

  it("renders installed status and 'Reinstall' action button and 'More' dropdown with null status and installed_versions", async () => {
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

    render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Installed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    const moreDropdown = getMoreDropdown();
    expect(moreDropdown).toBeEnabled();
  });

  it("renders failed status, 'Retry' button and hides 'More' dropdown with 'failed_install' and no installed versions detected", async () => {
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
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
    const moreText = screen.queryByText(/more/i);
    expect(moreText).not.toBeInTheDocument();
  });

  it("renders failed status and 'Install' action button and 'Retry uninstall' dropdown with 'failed_uninstall' status and installed_versions detected", async () => {
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
    const { user } = render(<SelfService {...TEST_PROPS} />);

    // waiting for the device software data to render
    await screen.findAllByText("test-software");

    expect(
      screen.getByTestId("install-status-cell__status--test")
    ).toHaveTextContent("Failed");

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeEnabled();
    const moreDropdown = getMoreDropdown();
    await user.click(moreDropdown);
    const dropdown = document.getElementById("react-select-9-listbox");
    if (!dropdown) {
      throw new Error("Could not find the dropdown actions");
    }
    const retryOption = within(dropdown).getByText(/Retry uninstall/i);
    expect(retryOption).toBeInTheDocument();
    expect(retryOption).toBeEnabled();
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
    const moreText = screen.queryByText(/more/i);
    expect(moreText).not.toBeInTheDocument();
  });

  it("renders installing status, disables Install action, and hides 'More' dropdown with 'pending_install' and no installed_version", async () => {
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
    const moreText = screen.queryByText(/more/i);
    expect(moreText).not.toBeInTheDocument();
  });

  it("renders uninstalling status and disables 'Reinstall' action button and 'More' dropdown with 'pending_uninstall'", async () => {
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

    expect(screen.getByRole("button", { name: "Reinstall" })).toBeDisabled(); // TODO: Should this say "Reinstall"?
    const moreDropdown = getMoreDropdown();
    expect(moreDropdown).toBeDisabled();
  });
});
