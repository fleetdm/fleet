import React from "react";
import { act, render, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "react-query";

import usePickerSearch from "./usePickerSearch";

// Probe component that exposes the hook's return values via the DOM
// so we can assert on them from tests.
interface IProbeProps<TResponse, TItem> {
  search: string;
  queryFn: (q: string) => Promise<TResponse>;
  selectItems: (data: TResponse | undefined) => TItem[];
}
const Probe = <TResponse, TItem>({
  search,
  queryFn,
  selectItems,
}: IProbeProps<TResponse, TItem>) => {
  const { items, isLoading, debouncedQuery } = usePickerSearch<
    TResponse,
    TItem
  >({
    search,
    queryKeyPrefix: ["test"],
    queryFn,
    selectItems,
  });
  return (
    <div>
      <span data-testid="loading">{String(isLoading)}</span>
      <span data-testid="debounced">{debouncedQuery}</span>
      <span data-testid="count">{items.length}</span>
    </div>
  );
};

const renderWithClient = (ui: React.ReactElement) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, cacheTime: 0 } },
  });
  const wrapper: React.FC<React.PropsWithChildren> = ({ children }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(ui, { wrapper });
};

describe("usePickerSearch", () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });
  afterEach(() => {
    jest.useRealTimers();
  });

  it("debounces the search prop before invoking queryFn", async () => {
    const queryFn = jest.fn().mockResolvedValue({ items: [] });
    const selectItems = (d?: { items: number[] }) => d?.items ?? [];

    const { rerender } = renderWithClient(
      <Probe search="a" queryFn={queryFn} selectItems={selectItems} />
    );

    // queryFn fires immediately with the *initial* debouncedQuery state
    // (the initial useState(search.trim()) value).
    await waitFor(() => expect(queryFn).toHaveBeenCalledWith("a"));

    // Update search rapidly — debounce should swallow intermediate values.
    rerender(<Probe search="ab" queryFn={queryFn} selectItems={selectItems} />);
    rerender(
      <Probe search="abc" queryFn={queryFn} selectItems={selectItems} />
    );

    // Before the timer fires, queryFn should still only have one call.
    expect(queryFn).toHaveBeenCalledTimes(1);

    // Advance past the debounce window.
    await act(async () => {
      jest.advanceTimersByTime(200);
    });

    await waitFor(() => {
      expect(queryFn).toHaveBeenCalledWith("abc");
    });
    // Intermediate "ab" should not have been queried.
    expect(queryFn).not.toHaveBeenCalledWith("ab");
  });

  it("trims the search input before debouncing", async () => {
    const queryFn = jest.fn().mockResolvedValue({ items: [] });
    const selectItems = (d?: { items: number[] }) => d?.items ?? [];

    renderWithClient(
      <Probe
        search="   padded   "
        queryFn={queryFn}
        selectItems={selectItems}
      />
    );

    await waitFor(() => expect(queryFn).toHaveBeenCalledWith("padded"));
  });

  it("uses selectItems to extract the displayed array from the response", async () => {
    const queryFn = jest
      .fn()
      .mockResolvedValue({ inner: { items: [1, 2, 3] } });
    const selectItems = (d?: { inner: { items: number[] } }) =>
      d?.inner?.items ?? [];

    const { getByTestId } = renderWithClient(
      <Probe search="" queryFn={queryFn} selectItems={selectItems} />
    );

    await waitFor(() => {
      expect(getByTestId("count").textContent).toBe("3");
    });
  });
});
