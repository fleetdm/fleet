import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { RouteComponentProps } from "react-router";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockLocation,
  createMockRouter,
} from "test/test-utils";

import ScriptBatchDetailsPage from "./ScriptBatchDetailsPage";

const BATCH_ID = "missing-batch-id";

const renderPage = () => {
  const render = createCustomRenderer({ withBackendMock: true });
  const props = ({
    router: createMockRouter(),
    routeParams: { batch_execution_id: BATCH_ID },
    location: createMockLocation({
      search: "?status=ran&page=0",
      query: { status: "ran", page: "0" },
    }),
  } as unknown) as RouteComponentProps<
    undefined,
    { batch_execution_id: string }
  >;
  return render(<ScriptBatchDetailsPage {...props} />);
};

describe("ScriptBatchDetailsPage", () => {
  it("shows an error instead of spinning forever when the batch no longer exists", async () => {
    mockServer.use(
      http.get(baseUrl(`/scripts/batch/${BATCH_ID}`), () =>
        HttpResponse.json(
          {
            message: "Not Found",
            errors: [{ name: "base", reason: "Not Found" }],
          },
          { status: 404 }
        )
      )
    );

    const { container } = renderPage();

    await waitFor(() => {
      expect(
        screen.getByText("Could not load script batch details.")
      ).toBeInTheDocument();
    });
    expect(container.querySelector(".loading-spinner")).not.toBeInTheDocument();
  });
});
