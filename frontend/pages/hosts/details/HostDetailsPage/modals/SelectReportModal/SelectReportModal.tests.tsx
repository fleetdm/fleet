import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  createCustomRenderer,
  createMockRouter,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";
import createMockUser from "__mocks__/userMock";

import SelectReportModal from "./SelectReportModal";

const mockRouter = createMockRouter();

const createReportsHandler = (reports: Record<string, unknown>[]) =>
  http.get(baseUrl("/reports"), () => {
    return HttpResponse.json({ queries: reports });
  });

const baseProps = {
  onCancel: jest.fn(),
  hostId: 1,
  hostTeamId: 1,
  router: mockRouter,
  currentTeamId: 1,
};

describe("SelectReportModal", () => {
  it("renders empty state with Create a report link when user can create", async () => {
    mockServer.use(createReportsHandler([]));
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser({ global_role: "admin" }),
        },
      },
    });

    render(<SelectReportModal {...baseProps} />);

    expect(await screen.findByText("No saved reports")).toBeInTheDocument();
    expect(screen.getByText("Create a report")).toBeInTheDocument();
    expect(screen.getByText(/to run\./)).toBeInTheDocument();
  });

  it("renders empty state without Create a report link for observer-only users", async () => {
    mockServer.use(createReportsHandler([]));
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isOnlyObserver: true,
          currentUser: createMockUser({ global_role: "observer" }),
        },
      },
    });

    render(<SelectReportModal {...baseProps} isOnlyObserver />);

    expect(await screen.findByText("No saved reports")).toBeInTheDocument();
    expect(
      screen.getByText("No reports are available to run.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Create a report")).not.toBeInTheDocument();
  });

  it("renders report list when reports exist", async () => {
    mockServer.use(
      createReportsHandler([
        {
          id: 1,
          name: "Test Report",
          description: "A test report",
          observer_can_run: true,
        },
      ])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser({ global_role: "admin" }),
        },
      },
    });

    render(<SelectReportModal {...baseProps} />);

    expect(await screen.findByText("Test Report")).toBeInTheDocument();
    expect(screen.getByText("A test report")).toBeInTheDocument();
  });

  it("renders description with create link for non-observer users", async () => {
    mockServer.use(
      createReportsHandler([
        { id: 1, name: "Report 1", observer_can_run: true },
      ])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser({ global_role: "admin" }),
        },
      },
    });

    render(<SelectReportModal {...baseProps} />);

    await screen.findByText("Report 1");
    expect(screen.getByText("create a report")).toBeInTheDocument();
  });

  it("renders description without create link for observer-only users", async () => {
    mockServer.use(
      createReportsHandler([
        { id: 1, name: "Report 1", observer_can_run: true },
      ])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isOnlyObserver: true,
          currentUser: createMockUser({ global_role: "observer" }),
        },
      },
    });

    render(<SelectReportModal {...baseProps} isOnlyObserver />);

    await screen.findByText("Report 1");
    expect(screen.queryByText("create a report")).not.toBeInTheDocument();
  });
});
