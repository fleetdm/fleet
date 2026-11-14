import { renderHook, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { getPatchDeadlineSettingsHandler } from "test/handlers/patch-deadline-settings-handlers";
import mockServer from "test/mock-server";

import usePatchDeadlineSettings from "./usePatchDeadlineSettings";

const buildWrapper = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
};

describe("usePatchDeadlineSettings", () => {
  it("fetches the new fleet's settings when fleetId changes", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const { result, rerender } = renderHook(
      ({ fleetId }: { fleetId: number }) => usePatchDeadlineSettings(fleetId),
      { wrapper: buildWrapper(), initialProps: { fleetId: 1 } }
    );

    await waitFor(() => expect(result.current.data?.fleet_id).toBe(1));

    rerender({ fleetId: 2 });

    await waitFor(() => expect(result.current.data?.fleet_id).toBe(2));
  });

  it("stays idle without a fleetId", () => {
    const { result } = renderHook(() => usePatchDeadlineSettings(undefined), {
      wrapper: buildWrapper(),
    });

    expect(result.current.isIdle).toBe(true);
  });
});
