import React from "react";
import { waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import queriesAPI, { IQueriesResponse } from "services/entities/queries";

import ReportPicker from "./ReportPicker";

jest.mock("services/entities/queries", () => ({
  __esModule: true,
  default: { loadAll: jest.fn() },
}));

const mockedQueries = queriesAPI as jest.Mocked<typeof queriesAPI>;

const renderPicker = createCustomRenderer({ withBackendMock: true });

// Minimal valid response — typed so a future IQueriesResponse rename
// would surface here instead of being hidden by `as any`.
const emptyQueriesResponse: IQueriesResponse = {
  queries: [],
  count: 0,
  inherited_query_count: 0,
  meta: { has_next_results: false, has_previous_results: false },
};

beforeEach(() => {
  mockedQueries.loadAll.mockReset();
  mockedQueries.loadAll.mockResolvedValue(emptyQueriesResponse);
});

describe("ReportPicker", () => {
  it("scopes by currentTeam when a real team is selected, with mergeInherited", async () => {
    renderPicker(
      <ReportPicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedQueries.loadAll).toHaveBeenCalledWith(
        expect.objectContaining({
          teamId: 5,
          mergeInherited: true,
          scope: "queries",
        })
      );
    });
  });

  it("passes teamId undefined when currentTeam is All fleets", async () => {
    renderPicker(
      <ReportPicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedQueries.loadAll).toHaveBeenCalledWith(
        expect.objectContaining({ teamId: undefined })
      );
    });
  });

  it("renders the fleet-scoped empty state", async () => {
    const { findByText } = renderPicker(
      <ReportPicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No reports found in Engineering\./)
    ).toBeInTheDocument();
  });

  it("renders the search-specific empty state with fleet label", async () => {
    const { findByText } = renderPicker(
      <ReportPicker
        search="missing"
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No reports match "missing" in Engineering\./)
    ).toBeInTheDocument();
  });

  it("renders 'in this fleet' empty state when Unassigned is selected", async () => {
    const { findByText } = renderPicker(
      <ReportPicker
        search=""
        currentTeam={{ id: 0, name: "Unassigned" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No reports found in this fleet\./)
    ).toBeInTheDocument();
  });

  it("renders a fleet-less empty state on All fleets", async () => {
    const { findByText } = renderPicker(
      <ReportPicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    // No suffix when context is All fleets.
    const node = await findByText(/^No reports found\.$/);
    expect(node).toBeInTheDocument();
  });
});
