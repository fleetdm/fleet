import React from "react";
import { screen, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";

import ManageHostsPage from "./ManageHostsPage";

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
      .getByText("Manage enroll secret")
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
});
