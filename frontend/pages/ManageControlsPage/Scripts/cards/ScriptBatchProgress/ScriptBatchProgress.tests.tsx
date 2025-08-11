import React from "react";
import { cleanup, screen } from "@testing-library/react";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
  waitForLoadingToFinish,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { http, HttpResponse } from "msw";

import { ScriptBatchStatus } from "interfaces/script";

import { createMockBatchScriptSummary } from "__mocks__/scriptMock";

import ScriptBatchProgress, {
  EMPTY_STATE_DETAILS,
} from "./ScriptBatchProgress";
import { ScriptsLocation } from "../../Scripts";

const emptyTeamBatchSummariesHandler = http.get(baseUrl("/scripts/batch"), () =>
  HttpResponse.json({
    batch_executions: [],
    meta: { has_next_results: false, has_previous_results: false },
    count: 0,
  })
);

const teamBatchSummariesHandler = http.get(
  baseUrl("/scripts/batch"),
  ({ params }) => {
    const { status } = params;
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
    // TODO if status param === "scheduled"

    // TODO if status param === "finished"
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
  it("Renders each tab and its empty state from URL navigation", async () => {
    mockServer.use(emptyTeamBatchSummariesHandler);

    await testTabURLNavAndEmpty("started");
    await testTabURLNavAndEmpty("scheduled");
    testTabURLNavAndEmpty("finished");
  });

  // it("Renders each tab with appropriate scripts list", async () => {
  //   mockServer.use(teamBatchSummariesHandler);

  //   const render = createCustomRenderer({
  //     withBackendMock: true,
  //   });

  //   const { container } = render(
  //     <ScriptBatchProgress
  //       router={createMockRouter()}
  //       teamId={1}
  //       location={getTestLocation("started")}
  //     />
  //   );

  //   await waitForLoadingToFinish(container);

  //   expect(screen.getByText("Test Script 1")).toBeInTheDocument();
  //   expect(screen.getByText("just now")).toBeInTheDocument();
  //   // (ran + errored) / targeted
  //   expect(screen.getByText("65 / 100")).toBeInTheDocument();
  //   expect(screen.getByText("hosts")).toBeInTheDocument();

  //   // TODO - click Scheduled, expect scheduled script summaries

  //   // TODO - click Finished, expect scheduled script summaries
  // });
});
