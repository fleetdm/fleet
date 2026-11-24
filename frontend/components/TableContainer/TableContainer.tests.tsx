import { render, screen, waitFor } from "@testing-library/react";
import React, { useState } from "react";

import TableContainer, { ITableQueryData } from "./TableContainer";

const COLUMN_CONFIGS = [
  {
    title: "Name",
    Header: "Name",
    accessor: "name",
    disableSortBy: true,
  },
];

const EmptyComponent = () => <div>No items found</div>;

// pageIndex requested by the most recent onQueryChange call.
const lastRequestedPageIndex = (onQueryChange: jest.Mock) => {
  const { calls } = onQueryChange.mock;
  return (calls[calls.length - 1][0] as ITableQueryData).pageIndex;
};

const PAGE_SIZE = 20;

// Simulates a real parent: pageIndex is URL-driven, and the data shown is
// derived from the requested page. Used to exercise the page-correction effect
// end-to-end (and guard against navigation feedback loops).
const ServerPaginatedTable = ({
  initialPage,
  totalCount,
}: {
  initialPage: number;
  totalCount: number;
}) => {
  const [page, setPage] = useState(initialPage);
  const start = page * PAGE_SIZE;
  const rows = Array.from({
    length: Math.max(0, Math.min(PAGE_SIZE, totalCount - start)),
  }).map((_, i) => ({ name: `row ${start + i}` }));

  return (
    <TableContainer
      columnConfigs={COLUMN_CONFIGS}
      data={rows}
      isLoading={false}
      emptyComponent={EmptyComponent}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      pageIndex={page}
      onQueryChange={(q: ITableQueryData) => setPage(q.pageIndex)}
      defaultSortHeader="name"
    />
  );
};

describe("TableContainer - server-side empty page", () => {
  it("navigates back to the last page with data when a non-first page is empty", async () => {
    const onQueryChange = jest.fn();

    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        pageIndex={1}
        totalCount={20}
        onQueryChange={onQueryChange}
        defaultSortHeader="name"
      />
    );

    // The empty page (index 1) is not a resting state: the table should request
    // the last page that actually has data (index 0 here).
    await waitFor(() => {
      expect(onQueryChange).toHaveBeenCalled();
      expect(lastRequestedPageIndex(onQueryChange)).toBe(0);
    });
  });

  it("shows the empty state and stays put on the first page", async () => {
    const onQueryChange = jest.fn();

    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        pageIndex={0}
        totalCount={0}
        onQueryChange={onQueryChange}
        defaultSortHeader="name"
      />
    );

    expect(await screen.findByText("No items found")).toBeInTheDocument();
    await waitFor(() => {
      expect(lastRequestedPageIndex(onQueryChange)).toBe(0);
    });
  });

  it("does not redirect while the page is still loading", () => {
    const onQueryChange = jest.fn();

    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        pageIndex={1}
        totalCount={20}
        onQueryChange={onQueryChange}
        defaultSortHeader="name"
      />
    );

    // While loading we can't know the page is truly empty, so the requested
    // page must never be corrected away from the one that was asked for.
    const requestedPageIndexes = onQueryChange.mock.calls.map(
      (call) => (call[0] as ITableQueryData).pageIndex
    );
    expect(requestedPageIndexes).not.toContain(0);
  });

  // Regression: entering an out-of-range page via the URL must settle on the
  // last page with data without looping.
  it("settles on the last page with data when entered on an out-of-range page", async () => {
    // 21 rows -> pages 0 (20 rows) and 1 (1 row); pages >= 2 are empty.
    render(<ServerPaginatedTable initialPage={3} totalCount={21} />);

    await waitFor(() => {
      expect(screen.getByText("row 20")).toBeInTheDocument();
    });
    expect(screen.queryByText("No items found")).not.toBeInTheDocument();
  });

  it("jumps straight to the first page when the total count is a known zero", async () => {
    const onQueryChange = jest.fn();

    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        pageIndex={3}
        totalCount={0}
        onQueryChange={onQueryChange}
        defaultSortHeader="name"
      />
    );

    await waitFor(() => {
      expect(onQueryChange).toHaveBeenCalled();
      expect(lastRequestedPageIndex(onQueryChange)).toBe(0);
    });
    // A known-empty count should jump straight to page 0, not step 3 -> 2 -> 1.
    const requestedPageIndexes = onQueryChange.mock.calls.map(
      (call) => (call[0] as ITableQueryData).pageIndex
    );
    expect(requestedPageIndexes).not.toContain(2);
    expect(requestedPageIndexes).not.toContain(1);
  });
});

describe("TableContainer - multi-column filter no-match overlay", () => {
  it("renders the overlay message and keeps the real header when a client filter yields zero rows", async () => {
    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[{ name: "alpha" }, { name: "beta" }]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSideFilter
        isMultiColumnFilter
        filters={{ name: "zzz-no-match" }}
        defaultSortHeader="name"
      />
    );

    expect(
      await screen.findByText("No items match the current search criteria")
    ).toBeInTheDocument();
    // The real header must still render so the per-column filter inputs stay
    // reachable — that's the whole reason isMultiColumnFilter exists.
    expect(screen.getByText("Name")).toBeInTheDocument();
    // The standard empty component must not fire in this path.
    expect(screen.queryByText("No items found")).not.toBeInTheDocument();
  });

  it("does not flash the overlay on the initial empty render before data arrives", () => {
    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSideFilter
        isMultiColumnFilter
        defaultSortHeader="name"
      />
    );

    // Zero data with no filter applied is a pre-data state, not a no-match
    // state. The overlay must stay hidden so it doesn't flash on first load.
    expect(
      screen.queryByText("No items match the current search criteria")
    ).not.toBeInTheDocument();
  });

  it("does not render the overlay message when isMultiColumnFilter is unset", () => {
    render(
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={[]}
        isLoading={false}
        emptyComponent={EmptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader="name"
      />
    );

    expect(
      screen.queryByText("No items match the current search criteria")
    ).not.toBeInTheDocument();
    expect(screen.getByText("No items found")).toBeInTheDocument();
  });
});
