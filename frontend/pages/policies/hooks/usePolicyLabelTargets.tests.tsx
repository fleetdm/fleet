import React from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "react-query";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import { AppContext, IAppContext, initialState } from "context/app";
import { createMockTeamSummary } from "__mocks__/teamMock";
import { ILabelPolicy, ILabelSummary } from "interfaces/label";
import labelsAPI from "services/entities/labels";

import usePolicyLabelTargets from "./usePolicyLabelTargets";

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

const mockLabels: ILabelSummary[] = [
  { id: 1, name: "Fun", description: "", label_type: "regular" },
  { id: 2, name: "Fresh", description: "", label_type: "regular" },
];

// Stored-policy label arrays must be referentially stable across renders (as
// they are in production, where they come from PolicyContext), so define them
// once rather than inline in the render callback.
const STORED_INCLUDE_ANY: ILabelPolicy[] = [{ id: 1, name: "Fun" }];
const STORED_INCLUDE_ALL: ILabelPolicy[] = [{ id: 1, name: "Fun" }];
const STORED_EXCLUDE_ANY: ILabelPolicy[] = [{ id: 2, name: "Fresh" }];
const STORED_EXCLUDE_ALL: ILabelPolicy[] = [{ id: 2, name: "Fresh" }];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () =>
  HttpResponse.json({ labels: mockLabels })
);

const buildWrapper = (appOverrides: Partial<IAppContext>) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, cacheTime: 0 } },
  });
  const value: IAppContext = { ...initialState, ...appOverrides };
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>
      <AppContext.Provider value={value}>{children}</AppContext.Provider>
    </QueryClientProvider>
  );
};

const premiumWrapper = (appOverrides: Partial<IAppContext> = {}) =>
  buildWrapper({
    isPremiumTier: true,
    currentTeam: createMockTeamSummary(),
    ...appOverrides,
  });

describe("usePolicyLabelTargets", () => {
  beforeEach(() => {
    mockServer.use(labelSummariesHandler);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("defaults to All hosts and an empty payload when no initial labels are given", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    expect(result.current.selectedTargetType).toBe("All hosts");
    expect(result.current.hasCustomLabels).toBe(false);
    expect(result.current.getLabelsPayload()).toEqual({
      labels_include_any: [],
      labels_include_all: [],
      labels_exclude_any: [],
      labels_exclude_all: [],
    });
  });

  it("fetches the team's custom labels and exposes them via selectorProps", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    await waitFor(() => {
      expect(result.current.selectorProps.labels.map((l) => l.name)).toEqual([
        "Fresh",
        "Fun",
      ]);
    });
  });

  it("does not fetch labels when no team is selected", () => {
    const summarySpy = jest.spyOn(labelsAPI, "summary");

    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: buildWrapper({ isPremiumTier: true, currentTeam: undefined }),
    });

    expect(summarySpy).not.toHaveBeenCalled();
    expect(result.current.selectorProps.labels).toEqual([]);
  });

  it("seeds Custom + include-any from a policy's stored include labels", async () => {
    const { result } = renderHook(
      () => usePolicyLabelTargets({ includeAny: STORED_INCLUDE_ANY }),
      { wrapper: premiumWrapper() }
    );

    await waitFor(() => {
      expect(result.current.selectedTargetType).toBe("Custom");
    });
    expect(result.current.hasCustomLabels).toBe(true);
    expect(result.current.selectorProps.includeConfig.mode).toBe("any");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_include_any: ["Fun"],
      labels_include_all: [],
      labels_exclude_any: [],
      labels_exclude_all: [],
    });
  });

  it("seeds the exclude-all scope when a policy stores exclude_all labels", async () => {
    const { result } = renderHook(
      () => usePolicyLabelTargets({ excludeAll: STORED_EXCLUDE_ALL }),
      { wrapper: premiumWrapper() }
    );

    await waitFor(() => {
      expect(result.current.selectedTargetType).toBe("Custom");
    });
    expect(result.current.selectorProps.excludeConfig.mode).toBe("all");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_exclude_all: ["Fresh"],
      labels_exclude_any: [],
      labels_include_any: [],
    });
  });

  it("builds the include-any payload after the user selects Custom and a label", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: true,
      })
    );

    expect(result.current.selectedTargetType).toBe("Custom");
    expect(result.current.hasCustomLabels).toBe(true);
    expect(result.current.getLabelsPayload().labels_include_any).toEqual([
      "Fun",
    ]);
  });

  it("clears the payload when the user switches back to All hosts", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: true,
      })
    );
    act(() => result.current.selectorProps.onSelectTargetType("All hosts"));

    expect(result.current.getLabelsPayload().labels_include_any).toEqual([]);
  });

  it("seeds Custom + include-all from a policy's stored include_all labels", async () => {
    const { result } = renderHook(
      () => usePolicyLabelTargets({ includeAll: STORED_INCLUDE_ALL }),
      { wrapper: premiumWrapper() }
    );

    await waitFor(() => {
      expect(result.current.selectedTargetType).toBe("Custom");
    });
    expect(result.current.hasCustomLabels).toBe(true);
    expect(result.current.selectorProps.includeConfig.mode).toBe("all");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_include_all: ["Fun"],
      labels_include_any: [],
      labels_exclude_any: [],
      labels_exclude_all: [],
    });
  });

  it("seeds Custom + exclude-any from a policy's stored exclude_any labels", async () => {
    const { result } = renderHook(
      () => usePolicyLabelTargets({ excludeAny: STORED_EXCLUDE_ANY }),
      { wrapper: premiumWrapper() }
    );

    await waitFor(() => {
      expect(result.current.selectedTargetType).toBe("Custom");
    });
    expect(result.current.hasCustomLabels).toBe(true);
    expect(result.current.selectorProps.excludeConfig.mode).toBe("any");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_exclude_any: ["Fresh"],
      labels_exclude_all: [],
      labels_include_any: [],
      labels_include_all: [],
    });
  });

  it("moves the include selection to labels_include_all when its mode switches to All", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: true,
      })
    );
    expect(result.current.getLabelsPayload().labels_include_any).toEqual([
      "Fun",
    ]);

    act(() => result.current.selectorProps.includeConfig.onSelectMode?.("all"));

    expect(result.current.selectorProps.includeConfig.mode).toBe("all");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_include_all: ["Fun"],
      labels_include_any: [],
    });
  });

  it("moves the exclude selection to labels_exclude_all when its mode switches to All", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.excludeConfig.onSelectLabel({
        name: "Fresh",
        value: true,
      })
    );
    expect(result.current.getLabelsPayload().labels_exclude_any).toEqual([
      "Fresh",
    ]);

    act(() => result.current.selectorProps.excludeConfig.onSelectMode?.("all"));

    expect(result.current.selectorProps.excludeConfig.mode).toBe("all");
    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_exclude_all: ["Fresh"],
      labels_exclude_any: [],
    });
  });

  it("removes a label from the payload when it is deselected", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: true,
      })
    );
    expect(result.current.hasCustomLabels).toBe(true);

    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: false,
      })
    );

    expect(result.current.hasCustomLabels).toBe(false);
    expect(result.current.getLabelsPayload().labels_include_any).toEqual([]);
  });

  it("builds a combined payload with both include and exclude selections", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    act(() => result.current.selectorProps.onSelectTargetType("Custom"));
    act(() =>
      result.current.selectorProps.includeConfig.onSelectLabel({
        name: "Fun",
        value: true,
      })
    );
    act(() =>
      result.current.selectorProps.excludeConfig.onSelectLabel({
        name: "Fresh",
        value: true,
      })
    );

    expect(result.current.getLabelsPayload()).toMatchObject({
      labels_include_any: ["Fun"],
      labels_exclude_any: ["Fresh"],
      labels_include_all: [],
      labels_exclude_all: [],
    });
  });

  it("reports a loading state while labels are being fetched", async () => {
    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    expect(result.current.selectorProps.isLoadingLabels).toBe(true);
    await waitFor(() => {
      expect(result.current.selectorProps.isLoadingLabels).toBe(false);
    });
  });

  it("surfaces an error state when the labels request fails", async () => {
    // A 4xx is terminal under DEFAULT_USE_QUERY_OPTIONS' retry policy, so the
    // error surfaces immediately (a 5xx would retry and outrun waitFor).
    mockServer.use(
      http.get(
        baseUrl("/labels/summary"),
        () => new HttpResponse(null, { status: 403 })
      )
    );

    const { result } = renderHook(() => usePolicyLabelTargets(), {
      wrapper: premiumWrapper(),
    });

    await waitFor(() => {
      expect(result.current.selectorProps.isErrorLabels).toBe(true);
    });
    expect(result.current.selectorProps.labels).toEqual([]);
  });
});
