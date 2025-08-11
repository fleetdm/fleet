import React from "react";
import { cleanup, screen, waitFor } from "@testing-library/react";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
  waitForLoadingToFinish,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import createMockConfig, { DEFAULT_LICENSE_MOCK } from "__mocks__/configMock";
import { IScriptBatchSummaryV2 } from "services/entities/scripts";
import { ScriptBatchStatus } from "interfaces/script";

import { createMockBatchScriptSummary } from "__mocks__/scriptMock";

import ScriptBatchProgress, {
  EMPTY_STATE_DETAILS,
} from "./ScriptBatchProgress";
import { ScriptsLocation } from "../../Scripts";

// mock batch summaries API
// const createMockBatchSummaries = (
//   status: ScriptBatchStatus,
//   count = 3
// ): IScriptBatchSummaryV2[] => {
//   const now = new Date().toISOString();
//   const future = new Date(Date.now() + 86400000).toISOString(); // 24 hours in the future

//   return Array(count)
//     .fill(null)
//     .map((_, i) => {
//       let ranCount = 0;
//       let errorCount = 0;

//       if (status === "started") {
//         ranCount = 5;
//         errorCount = 1;
//       } else if (status === "finished") {
//         ranCount = 8;
//         errorCount = 2;
//       }

//       return {
//         batch_execution_id: `batch-${status}-${i}`,
//         script_name: `Test Script ${i + 1}`,
//         status,
//         targeted_host_count: 10,
//         ran_host_count: ranCount,
//         errored_host_count: errorCount,
//         pending_host_count: 10 - ranCount - errorCount,
//         incompatible_host_count: 0,
//         canceled_host_count: 0,
//         started_at: status === "started" || status === "finished" ? now : null,
//         finished_at: status === "finished" ? now : null,
//         not_before: status === "scheduled" ? future : null,
//         canceled: false,
//         created_at: now,
//         script_id: i + 1,
//         team_id: 1,
//         user_id: 1,
//       };
//     });
// };

// const createBatchSummariesHandler = (status: ScriptBatchStatus, count = 3) => {
//   return handlers.rest.get(
//     "/api/v1/fleet/scripts/run/batch",
//     (req: any, res: any, ctx: any) => {
//       const queryStatus = req.url.searchParams.get("status");

//       if (queryStatus === status) {
//         return res(
//           ctx.json({
//             count,
//             batch_executions: createMockBatchSummaries(status, count),
//           })
//         );
//       }

//       return res(
//         ctx.json({
//           count: 0,
//           batch_executions: [],
//         })
//       );
//     }
//   );
// };

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
            status: "started",
            finished_at: null,
          }),
        ],
        meta: { has_next_results: false, has_previous_results: false },
        count: 1,
      });
    }
    // if status param === "scheduled"
    // if status param === "finished"
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
  it.only("Renders each tab and its empty state from URL navigation", async () => {
    mockServer.use(emptyTeamBatchSummariesHandler);

    await testTabURLNavAndEmpty("started");
    await testTabURLNavAndEmpty("scheduled");
    testTabURLNavAndEmpty("finished");
  });

  it("Renders each tab with appropriate scripts list", async () => {
    const location: ScriptsLocation = {
      pathname: "/controls/scripts/batch-progress",
      query: { status: "started" },
      search: "?status=started",
    };

    const { unmount, rerender } = render(
      <ScriptBatchProgress
        router={createMockRouter()}
        teamId={1}
        location={location}
      />
    );

    // expect started scripts to be listed, each with started at text and the right ratio of hosts responses
    await waitFor(() => {
      expect(screen.getByText("3 batch scripts")).toBeInTheDocument();
      expect(screen.getAllByText(/Test Script \d/).length).toBe(3);
      expect(screen.getAllByText(/Started/).length).toBe(3);
      expect(screen.getAllByText(/5 \/ 10 hosts/).length).toBe(3);
      expect(screen.getAllByText("1")).toHaveLength(3); // Error count
    });

    unmount();

    // render the "Scheduled" tab
    const scheduledRouter = createMockRouter();

    const scheduledLocation: ScriptsLocation = {
      pathname: "/controls/scripts/batch-progress",
      query: { status: "scheduled" },
      search: "?status=scheduled",
    };

    rerender(
      <ScriptBatchProgress
        router={scheduledRouter}
        teamId={1}
        location={scheduledLocation}
      />
    );

    // expect scheduled scripts to be listed, each with will run at text NO host stats / progress bar
    await waitFor(() => {
      expect(screen.getByText("2 batch scripts")).toBeInTheDocument();
      expect(screen.getAllByText(/Test Script \d/).length).toBe(2);
      expect(screen.getAllByText(/Scheduled to start in/).length).toBe(2);
      // Verify no progress bars are shown for scheduled scripts
      expect(screen.queryByText(/hosts/)).not.toBeInTheDocument();
    });

    unmount();

    // render the "Finished" tab
    const finishedRouter = createMockRouter();

    const finishedLocation: ScriptsLocation = {
      pathname: "/controls/scripts/batch-progress",
      query: { status: "finished" },
      search: "?status=finished",
    };

    rerender(
      <ScriptBatchProgress
        router={finishedRouter}
        teamId={1}
        location={finishedLocation}
      />
    );

    // expect finished scripts to be listed, each with finished at text, host stats, and progress bar
    await waitFor(() => {
      expect(screen.getByText("4 batch scripts")).toBeInTheDocument();
      expect(screen.getAllByText(/Test Script \d/).length).toBe(4);
      expect(screen.getAllByText(/Completed/).length).toBe(4);
      expect(screen.getAllByText(/10 \/ 10 hosts/).length).toBe(4);
      expect(screen.getAllByText("2")).toHaveLength(4); // Error count
    });
  });
});
