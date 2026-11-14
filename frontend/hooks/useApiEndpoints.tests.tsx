import { renderHook, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { IApiEndpoint } from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

import useApiEndpoints from "./useApiEndpoints";

jest.mock("services/entities/api_endpoints");

const LIST_HOSTS: IApiEndpoint = {
  method: "GET",
  path: "/api/v1/fleet/hosts",
  display_name: "List hosts",
  deprecated: false,
};

const buildWrapper = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
};

describe("useApiEndpoints", () => {
  afterEach(() => {
    (apiEndpointsAPI.loadAll as jest.Mock).mockReset();
  });

  it("returns the endpoint catalog", async () => {
    (apiEndpointsAPI.loadAll as jest.Mock).mockResolvedValue([LIST_HOSTS]);
    const { result } = renderHook(() => useApiEndpoints(), {
      wrapper: buildWrapper(),
    });

    await waitFor(() => expect(result.current.data).toEqual([LIST_HOSTS]));
  });

  it("shares one request between consumers", async () => {
    (apiEndpointsAPI.loadAll as jest.Mock).mockResolvedValue([LIST_HOSTS]);
    const { result } = renderHook(
      () => ({ first: useApiEndpoints(), second: useApiEndpoints() }),
      { wrapper: buildWrapper() }
    );

    await waitFor(() =>
      expect(result.current.first.data).toEqual([LIST_HOSTS])
    );
    expect(result.current.second.data).toEqual([LIST_HOSTS]);
    expect(apiEndpointsAPI.loadAll).toHaveBeenCalledTimes(1);
  });

  it("stays idle when disabled", () => {
    const { result } = renderHook(() => useApiEndpoints({ enabled: false }), {
      wrapper: buildWrapper(),
    });

    expect(result.current.isIdle).toBe(true);
    expect(apiEndpointsAPI.loadAll).not.toHaveBeenCalled();
  });
});
