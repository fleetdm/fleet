import React from "react";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";

import ManageHostsPage from "./ManageHostsPage";

// Exporting hosts wraps the CSV response in a File and hands it to
// FileSaver.saveAs, which is a no-op in jsdom — mock it out so the click
// handler completes without touching the DOM download machinery.
jest.mock("file-saver");

const mockAppContext = {
  isGlobalAdmin: true,
  isGlobalMaintainer: false,
  isOnGlobalTeam: true,
  isOnlyObserver: false,
  isPremiumTier: false,
  isFreeTier: true,
  currentUser: createMockUser({ global_role: "admin" }),
  config: createMockConfig(),
  setFilteredHostsPath: jest.fn(),
  setFilteredPoliciesPath: jest.fn(),
  setFilteredQueriesPath: jest.fn(),
  setFilteredSoftwarePath: jest.fn(),
};

// Handlers

const getConfigHandler = () =>
  http.get(baseUrl("/config"), () => {
    return HttpResponse.json(createMockConfig());
  });

const getLabelsHandler = () =>
  http.get(baseUrl("/labels"), () => {
    return HttpResponse.json({ labels: [] });
  });

const getHostsHandler = (hosts: Record<string, unknown>[] = []) =>
  http.get(baseUrl("/hosts"), () => {
    return HttpResponse.json({ hosts });
  });

const getHostsCountHandler = (count: number) =>
  http.get(baseUrl("/hosts/count"), () => {
    return HttpResponse.json({ count });
  });

const getGlobalEnrollSecretsHandler = () =>
  http.get(baseUrl("/spec/enroll_secret"), () => {
    return HttpResponse.json({
      spec: { secrets: [{ secret: "test-secret" }] },
    });
  });

const getMeHandler = () =>
  http.get(baseUrl("/me"), () => {
    return HttpResponse.json({
      user: createMockUser({ global_role: "admin" }),
    });
  });

// Mock props

interface IMockPropsOverrides {
  location?: {
    pathname?: string;
    search?: string;
    hash?: string;
    query?: Record<string, string>;
  };
  [key: string]: unknown;
}

const createMockProps = (overrides?: IMockPropsOverrides) => ({
  route: { path: "hosts/manage" },
  router: {
    push: jest.fn(),
    replace: jest.fn(),
    goBack: jest.fn(),
    goForward: jest.fn(),
    go: jest.fn(),
    setRouteLeaveHook: jest.fn(),
    isActive: jest.fn(),
    createHref: jest.fn(),
    createPath: jest.fn(),
  },
  params: {},
  location: {
    pathname: "/hosts/manage",
    search: "",
    hash: "",
    query: {},
    ...overrides?.location,
  },
  ...overrides,
});

const setupHandlers = (
  hostCount: number,
  hosts: Record<string, unknown>[] = []
) => {
  mockServer.use(
    getConfigHandler(),
    getLabelsHandler(),
    getHostsHandler(hosts),
    getHostsCountHandler(hostCount),
    getGlobalEnrollSecretsHandler(),
    getMeHandler()
  );
};

describe("ManageHostsPage", () => {
  it("renders truly empty state with disabled controls", async () => {
    setupHandlers(0);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: mockAppContext },
    });

    render(<ManageHostsPage {...(createMockProps() as any)} />);

    // Empty state copy
    expect(await screen.findByText("No hosts")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Fleet refers to computers, servers, and mobile devices as hosts/
      )
    ).toBeInTheDocument();

    // Host count
    expect(screen.getByText("0 hosts")).toBeInTheDocument();

    // Controls are disabled
    expect(
      screen.getByRole("button", { name: /export hosts/i })
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /edit columns/i })
    ).toBeDisabled();
    expect(screen.getByPlaceholderText(/search name/i)).toBeDisabled();

    // Add hosts button still visible in the page header
    const headerWrap = screen
      .getByText("Enroll secrets")
      .closest(".manage-hosts__button-wrap");
    expect(
      within(headerWrap as HTMLElement).getByText("Add hosts")
    ).toBeInTheDocument();
  });

  it("renders filtered empty state with enabled controls", async () => {
    setupHandlers(0);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: mockAppContext },
    });

    const props = createMockProps({
      location: {
        pathname: "/hosts/manage",
        search: "?query=nonexistent",
        hash: "",
        query: { query: "nonexistent" },
      },
    });

    render(<ManageHostsPage {...(props as any)} />);

    // Filtered empty state copy
    expect(
      await screen.findByText("No hosts match your filters")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Recently enrolled hosts will appear here after their first check-in/
      )
    ).toBeInTheDocument();

    // Controls are NOT disabled
    expect(screen.getByPlaceholderText(/search name/i)).not.toBeDisabled();
    expect(
      screen.getByRole("button", { name: /edit columns/i })
    ).not.toBeDisabled();
  });

  it("renders filtered empty state for query param filters with enabled controls", async () => {
    setupHandlers(0);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: mockAppContext },
    });

    const props = createMockProps({
      location: {
        pathname: "/hosts/manage",
        search: "?low_disk_space=32",
        hash: "",
        query: { low_disk_space: "32" },
      },
    });

    render(<ManageHostsPage {...(props as any)} />);

    // Filtered empty state copy (not the truly empty state)
    expect(
      await screen.findByText("No hosts match your filters")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Recently enrolled hosts will appear here after their first check-in/
      )
    ).toBeInTheDocument();

    // Controls are NOT disabled
    expect(screen.getByPlaceholderText(/search name/i)).not.toBeDisabled();
    expect(
      screen.getByRole("button", { name: /edit columns/i })
    ).not.toBeDisabled();
  });

  it("renders populated state with enabled controls", async () => {
    const mockHost = {
      id: 1,
      hostname: "test-host",
      display_name: "test-host",
      display_text: "test-host",
      status: "online",
      platform: "darwin",
      os_version: "macOS 14.0",
      team_id: null,
      team_name: null,
      primary_ip: "192.168.1.1",
      primary_mac: "00:00:00:00:00:00",
      seen_time: "2024-01-01T00:00:00Z",
      hardware_serial: "ABC123",
      computer_name: "test-host",
      cpu_type: "x86_64",
      memory: 8000000000,
      issues: { total_issues_count: 0, failing_policies_count: 0 },
    };

    setupHandlers(1, [mockHost]);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: mockAppContext },
    });

    render(<ManageHostsPage {...(createMockProps() as any)} />);

    expect(await screen.findByText("1 host")).toBeInTheDocument();

    // Controls are NOT disabled
    expect(screen.getByPlaceholderText(/search name/i)).not.toBeDisabled();
    expect(
      screen.getByRole("button", { name: /export hosts/i })
    ).not.toBeDisabled();
    expect(
      screen.getByRole("button", { name: /edit columns/i })
    ).not.toBeDisabled();
  });

  it("substitutes the display-only 'agent' column with real fields when exporting to CSV", async () => {
    const mockHost = {
      id: 1,
      hostname: "test-host",
      display_name: "test-host",
      display_text: "test-host",
      status: "online",
      platform: "darwin",
      os_version: "macOS 14.0",
      team_id: null,
      team_name: null,
      seen_time: "2024-01-01T00:00:00Z",
      issues: { total_issues_count: 0, failing_policies_count: 0 },
    };

    // Capture the `columns` query param sent to the hosts report endpoint. The
    // "agent" column is a display-only column (it coalesces orbit and osquery
    // versions) with no corresponding CSV field on the backend, so the request
    // must send the underlying fields instead — otherwise the backend rejects
    // it with "invalid column name" and the export fails (#47085).
    let exportedColumns: string | null = null;
    setupHandlers(1, [mockHost]);
    mockServer.use(
      http.get(baseUrl("/hosts/report"), ({ request }) => {
        exportedColumns = new URL(request.url).searchParams.get("columns");
        return HttpResponse.text("hostname\ntest-host\n");
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: mockAppContext },
    });

    const user = userEvent.setup();
    render(<ManageHostsPage {...(createMockProps() as any)} />);

    expect(await screen.findByText("1 host")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /export hosts/i }));

    await waitFor(() => expect(exportedColumns).not.toBeNull());

    const columns = ((exportedColumns as unknown) as string).split(",");
    // The derived "agent" column is replaced by the real CSV fields...
    expect(columns).toContain("orbit_version");
    expect(columns).toContain("osquery_version");
    // ...and neither "agent" nor the display-only "selection" column is sent.
    expect(columns).not.toContain("agent");
    expect(columns).not.toContain("selection");
  });
});
