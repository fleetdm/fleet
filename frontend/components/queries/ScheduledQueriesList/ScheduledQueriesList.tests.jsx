import React from "react";
import { render, screen } from "@testing-library/react";

import { scheduledQueryStub } from "test/stubs";
import ScheduledQueriesList from "./index";

const scheduledQueries = [
  { ...scheduledQueryStub, id: 1 },
  { ...scheduledQueryStub, id: 2 },
];

const defaultProps = {
  onCheckAllQueries: jest.fn(),
  onCheckQuery: jest.fn(),
  onDblClickQuery: jest.fn(),
  checkedScheduledQueryIDs: [],
};

describe("ScheduledQueriesList - component", () => {
  it("renders a ScheduledQueriesListItem for each scheduled query", () => {
    const { container } = render(
      <ScheduledQueriesList
        {...defaultProps}
        allQueries={[]}
        onHidePackForm={jest.fn()}
        onSelectQuery={jest.fn()}
        scheduledQueries={scheduledQueries}
        selectedScheduledQueryIDs={[]}
      />
    );
    screen.debug();
    expect(
      container.querySelectorAll(".scheduled-query-list-item").length
    ).toEqual(2);
  });

  it('renders "No queries found" help text when scheduled queries are available but the scheduled queries are filtered out', () => {
    render(
      <ScheduledQueriesList
        {...defaultProps}
        allQueries={[]}
        isScheduledQueriesAvailable
        onHidePackForm={jest.fn()}
        onSelectQuery={jest.fn()}
        scheduledQueries={[]}
        selectedScheduledQueryIDs={[]}
      />
    );

    expect(
      screen.getByText("No queries matched your search criteria.")
    ).toBeInTheDocument();
  });

  it("renders initial help text when no queries have been scheduled yet", () => {
    render(
      <ScheduledQueriesList
        allQueries={[]}
        onHidePackForm={jest.fn()}
        onSelectQuery={jest.fn()}
        scheduledQueries={[]}
        selectedScheduledQueryIDs={[]}
      />
    );

    expect(screen.getByText("Your pack is empty.")).toBeInTheDocument();
  });
});
