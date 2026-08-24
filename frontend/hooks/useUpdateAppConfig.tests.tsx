import React from "react";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "react-query";

import { AppContext, IAppContext, initialState } from "context/app";
import createMockConfig from "__mocks__/configMock";

import useUpdateAppConfig from "./useUpdateAppConfig";

const buildWrapper = (appOverrides: Partial<IAppContext> = {}) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const value: IAppContext = { ...initialState, ...appOverrides };
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>
      <AppContext.Provider value={value}>{children}</AppContext.Provider>
    </QueryClientProvider>
  );
  return { Wrapper, client };
};

describe("useUpdateAppConfig", () => {
  // Regression guard for #42546: this hook is the single primitive that pages
  // call after a config write to prime the cache directly from the write
  // response, avoiding a stale refetch under DB read-replica lag. Fanning both
  // the AppContext and the ["config"] React Query cache must stay in one call.
  it("writes the config to both AppContext and the React Query cache", () => {
    const setConfig = jest.fn();
    const { Wrapper, client } = buildWrapper({ setConfig });
    const updated = createMockConfig({
      org_info: {
        org_name: "New name",
        org_logo_url: "",
        org_logo_url_light_background: "",
        contact_url: "https://fleetdm.com/company/contact",
      },
    });

    const { result } = renderHook(() => useUpdateAppConfig(), {
      wrapper: Wrapper,
    });
    result.current(updated);

    expect(setConfig).toHaveBeenCalledTimes(1);
    expect(setConfig).toHaveBeenCalledWith(updated);
    expect(client.getQueryData(["config"])).toBe(updated);
  });

  it("returns a stable callback across renders when dependencies do not change", () => {
    const setConfig = jest.fn();
    const { Wrapper } = buildWrapper({ setConfig });

    const { result, rerender } = renderHook(() => useUpdateAppConfig(), {
      wrapper: Wrapper,
    });
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });
});
