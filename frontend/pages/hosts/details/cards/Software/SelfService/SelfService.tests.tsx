import React from "react";
import { screen, within, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { noop } from "lodash";
import {
  createCustomRenderer,
  createMockRouter,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import {
  createMockDeviceSoftware,
  createMockDeviceSoftwareResponse,
} from "__mocks__/deviceUserMock";
import {
  DEFAULT_INSTALLED_VERSION,
  DEFAULT_HOST_HOSTNAME,
  createMockHostSoftwarePackage,
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
  mdmEnrollmentStatus: "Off",
};

describe("SelfService", () => {
  it("should render the self service items correctly", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({ id: 1, name: "test1" }),
          createMockDeviceSoftware({ id: 2, name: "test2" }),
          createMockDeviceSoftware({ id: 3, name: "test3" }),
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

  it("renders installed status and 'Install' action button and 'Retry uninstall' dropdown with 'failed_uninstall' API status and installed_versions detected", async () => {
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
    ).toHaveTextContent("Installed");

    const moreDropdown = getMoreDropdown();
    await user.click(moreDropdown);
    // react-select generates instance-numbered listbox IDs; read aria-controls
    // off the combobox so this stays stable as more react-select instances are
    // added/removed elsewhere on the page.
    const listboxId = moreDropdown.getAttribute("aria-controls");
    const dropdown = listboxId ? document.getElementById(listboxId) : null;
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

  it("renders the self-service list for BYOD Account-Driven User Enrollment on mobile view", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({ id: 1, name: "user-enrolled-app" }),
        ],
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });

    render(
      <SelfService
        {...TEST_PROPS}
        isMobileView
        mdmEnrollmentStatus="On (personal)"
      />
    );

    // The "not supported" gate has been removed; the user-enrolled host gets
    // the same self-service list as a manually-enrolled iOS/iPadOS host.
    expect(await screen.findByText("user-enrolled-app")).toBeInTheDocument();
    expect(
      screen.queryByText(/Self-service isn't supported/i)
    ).not.toBeInTheDocument();
  });

  // After a (bulk) update finishes, the host-details refetch surfaces the app as
  // installed while its software inventory is still stale (installer version is
  // still newer than installed_versions). The "Updated" state should hold
  // throughout that refetch window — the "Update" button must not reappear.
  it("keeps the 'Updated' state (not the 'Update' button) while inventory refetch is pending after an update", async () => {
    const LAST_INSTALL_AT = "2022-01-01T12:00:00Z";
    // Host software inventory timestamp is newer than the last install, so the
    // stale-inventory app would otherwise resolve to "update_available".
    const HOST_SOFTWARE_UPDATED_AT = "2022-06-01T12:00:00Z";

    const makeUpdatableSoftware = (status: "installed" | "pending_install") =>
      createMockDeviceSoftware({
        id: 1,
        name: "test-update",
        status,
        // Installed version (1.0.0) is older than the packaged installer (2.0.0).
        installed_versions: [
          { ...DEFAULT_INSTALLED_VERSION, version: "1.0.0" },
        ],
        software_package: createMockHostSoftwarePackage({
          version: "2.0.0",
          last_install: {
            install_uuid: "abc-123",
            installed_at: LAST_INSTALL_AT,
          },
        }),
      });

    const softwareByPhase = {
      available: makeUpdatableSoftware("installed"),
      pending: makeUpdatableSoftware("pending_install"),
      // Update finished, but inventory still reports the old version.
      completed: makeUpdatableSoftware("installed"),
    };
    let phase: keyof typeof softwareByPhase = "available";

    mockServer.use(
      http.get(baseUrl("/device/:token/software"), () =>
        HttpResponse.json(
          createMockDeviceSoftwareResponse({
            software: [softwareByPhase[phase]],
            count: 1,
          })
        )
      ),
      http.post(baseUrl("/device/:token/software/install/:id"), () =>
        HttpResponse.json({})
      )
    );

    const render = createCustomRenderer({ withBackendMock: true });
    const { user, rerender } = render(
      <SelfService
        {...TEST_PROPS}
        hostSoftwareUpdatedAt={HOST_SOFTWARE_UPDATED_AT}
      />
    );

    // Initial state: an update is available, so the Updates card shows "Update".
    const getUpdatesCard = () =>
      screen.getByText("Updates").closest(".updates-card") as HTMLElement;
    await screen.findByText("Updates");
    expect(
      within(getUpdatesCard()).getByRole("button", { name: /^Update$/ })
    ).toBeEnabled();

    // User triggers the update; the dedicated poll observes it go pending.
    phase = "pending";
    await user.click(screen.getByRole("button", { name: "Update all" }));
    await within(getUpdatesCard()).findByText(/Updating/);

    // Host-details polling completes (the automatic refetch in the bug report),
    // which refetches self-service data and surfaces the completed-but-stale app.
    phase = "completed";
    rerender(
      <SelfService
        {...TEST_PROPS}
        hostSoftwareUpdatedAt={HOST_SOFTWARE_UPDATED_AT}
        isHostDetailsPolling
      />
    );
    rerender(
      <SelfService
        {...TEST_PROPS}
        hostSoftwareUpdatedAt={HOST_SOFTWARE_UPDATED_AT}
        isHostDetailsPolling={false}
      />
    );

    // The card must show "Updated" and must NOT fall back to an "Update" button.
    await waitFor(() => {
      expect(within(getUpdatesCard()).getByText("Updated")).toBeInTheDocument();
    });
    expect(
      within(getUpdatesCard()).queryByRole("button", { name: /^Update$/ })
    ).not.toBeInTheDocument();
  });
});
