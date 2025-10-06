import React from "react";
import { cleanup, screen, waitFor } from "@testing-library/react";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { http, HttpResponse } from "msw";

import { ScriptBatchStatus } from "interfaces/script";

import { createMockBatchScriptSummary } from "__mocks__/scriptMock";

import ScriptBatchProgress, {
  EMPTY_STATE_DETAILS,
} from "./ScriptBatchProgress";
import { ScriptsLocation } from "../../Scripts";

const waitForLoadingToFinish = async (container: HTMLElement) => {
  await waitFor(() => {
    expect(
      container.querySelector(".script-batch-progress__loading")
    ).not.toBeInTheDocument();
  });
};

const emptyTeamBatchSummariesHandler = http.get(baseUrl("/scripts/batch"), () =>
  HttpResponse.json({
    batch_executions: [],
    meta: { has_next_results: false, has_previous_results: false },
    count: 0,
  })
);

const teamBatchSummariesHandler = http.get(
  baseUrl("/scripts/batch"),
  ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get("status");
    if (status === "started") {
      return HttpResponse.json({
        batch_executions: [
          createMockBatchScriptSummary({
            script_name: "Test Script 1",
            status: "started",
            finished_at: null,
            started_at: new Date().toISOString(),
          }),
        ],
        meta: { has_next_results: false, has_previous_results: false },
        count: 1,
      });
    }
    if (status === "scheduled") {
      return HttpResponse.json({
        batch_executions: [
          createMockBatchScriptSummary({
            script_name: "Test Script 1",
            status: "scheduled",
            finished_at: null,
            not_before: "9999-01-01T10:11:00.000Z",
          }),
        ],
        meta: { has_next_results: false, has_previous_results: false },
        count: 1,
      });
    }
    if (status === "finished") {
      return HttpResponse.json({
        batch_executions: [
          createMockBatchScriptSummary({
            script_name: "Test Script 1",
            status: "finished",
            finished_at: "2025-07-01T10:00:00Z",
          }),
          createMockBatchScriptSummary({
            script_name: "Test Script 2",
            status: "finished",
            canceled: true,
            finished_at: "2025-06-02T11:12:00Z",
            targeted_host_count: 50,
            ran_host_count: 5,
            pending_host_count: 0,
            errored_host_count: 15,
            incompatible_host_count: 5,
            canceled_host_count: 25,
          }),
        ],
        meta: { has_next_results: false, has_previous_results: false },
        count: 1,
      });
    }

    return HttpResponse.json({});
  }
);

const getTestLocation = (status: ScriptBatchStatus): ScriptsLocation => ({
  pathname: "/controls/scripts/batch-progress",
  query: { status },
  search: `?status=${status}`,
});

const testTabURLNavAndEmpty = async (status: ScriptBatchStatus) => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });
  const { container } = render(
    <ScriptBatchProgress
      router={createMockRouter()}
      teamId={1}
      location={getTestLocation(status)}
    />
  );

  expect(
    screen.getByRole("tab", { name: "Started", selected: status === "started" })
  ).toBeInTheDocument();
  expect(
    screen.getByRole("tab", {
      name: "Scheduled",
      selected: status === "scheduled",
    })
  ).toBeInTheDocument();
  expect(
    screen.getByRole("tab", {
      name: "Finished",
      selected: status === "finished",
    })
  ).toBeInTheDocument();

  await waitForLoadingToFinish(container);

  expect(screen.getByText(EMPTY_STATE_DETAILS[status])).toBeInTheDocument();
  cleanup();
};

describe("ScriptBatchProgress", () => {
  it("Renders 'started' empty state from URL navigation", async () => {
    mockServer.use(emptyTeamBatchSummariesHandler);
    await testTabURLNavAndEmpty("started");
  });

  it("Renders 'scheduled' empty state from URL navigation", async () => {
    mockServer.use(emptyTeamBatchSummariesHandler);
    await testTabURLNavAndEmpty("scheduled");
  });

  it("Renders 'finished' empty state from URL navigation", async () => {
    mockServer.use(emptyTeamBatchSummariesHandler);
    await testTabURLNavAndEmpty("finished");
  });

  it("Renders the 'started' tab with appropriate scripts list", async () => {
    mockServer.use(teamBatchSummariesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { container } = render(
      <ScriptBatchProgress
        router={createMockRouter()}
        teamId={1}
        location={getTestLocation("started")}
      />
    );

    await waitForLoadingToFinish(container);

    await waitFor(() => {
      expect(screen.getByText("Test Script 1")).toBeInTheDocument();
      expect(screen.getByText("less than a minute ago")).toBeInTheDocument();
      // (ran + errored) / targeted
      expect(screen.getByText(/65\s+\/\s+100/m)).toBeInTheDocument();
      expect(screen.getByText(/hosts/)).toBeInTheDocument();
    });
  });

  it("Renders the 'scheduled' tab with appropriate scripts list", async () => {
    mockServer.use(teamBatchSummariesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { container } = render(
      <ScriptBatchProgress
        router={createMockRouter()}
        teamId={1}
        location={getTestLocation("scheduled")}
      />
    );

    await waitForLoadingToFinish(container);

    await waitFor(() => {
      expect(screen.getByText("Test Script 1")).toBeInTheDocument();
      expect(screen.getByText(/Will start/)).toBeInTheDocument();
      expect(screen.getByText(/in (about|over) \d+ years/)).toBeInTheDocument();
    });
  });

  it("Renders the 'finished' tab with appropriate scripts list", async () => {
    mockServer.use(teamBatchSummariesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { container } = render(
      <ScriptBatchProgress
        router={createMockRouter()}
        teamId={1}
        location={getTestLocation("finished")}
      />
    );

    await waitForLoadingToFinish(container);

    // Not a 100% awesome test because we're not correlating the script names
    // with the summaries.
    await waitFor(() => {
      expect(screen.getByText(/Test Script 1/)).toBeInTheDocument();
      expect(screen.getByText(/Completed/)).toBeInTheDocument();
      expect(screen.getByText(/65\s+\/\s+100/m)).toBeInTheDocument();
      expect(screen.getByText("Test Script 2")).toBeInTheDocument();
      expect(screen.getByText(/Canceled/)).toBeInTheDocument();
      expect(screen.getByText(/20\s+\/\s+50/m)).toBeInTheDocument();
    });
  });
});
