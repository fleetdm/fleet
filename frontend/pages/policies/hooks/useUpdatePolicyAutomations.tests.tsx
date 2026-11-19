import { act, renderHook, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { IPolicy } from "interfaces/policy";
import policiesAPI from "services/entities/policies";

import useUpdatePolicyAutomations from "./useUpdatePolicyAutomations";

const policy = { id: 7 } as IPolicy;

const setup = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  client.setQueryData(["config"], { webhook_settings: {} });
  client.setQueryData(["teams", 1], { fleet: { id: 1 } });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return { client, wrapper };
};

describe("useUpdatePolicyAutomations", () => {
  afterEach(() => jest.restoreAllMocks());

  it("saves the automation and marks the cached settings stale", async () => {
    const updateSpy = jest
      .spyOn(policiesAPI, "update")
      .mockResolvedValue({} as never);
    const onSuccess = jest.fn();
    const { client, wrapper } = setup();
    const { result } = renderHook(
      () => useUpdatePolicyAutomations({ policy, onSuccess }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        policyUpdate: { automation_type: "webhook_or_ticket" },
      });
    });

    await waitFor(() => expect(onSuccess).toHaveBeenCalled());
    expect(updateSpy).toHaveBeenCalledWith(7, {
      automation_type: "webhook_or_ticket",
    });
    expect(client.getQueryState(["config"])?.isInvalidated).toBe(true);
    expect(client.getQueryState(["teams", 1])?.isInvalidated).toBe(true);
  });

  it("leaves the cached settings alone when the save fails", async () => {
    jest.spyOn(policiesAPI, "update").mockRejectedValue(new Error("nope"));
    const onError = jest.fn();
    const { client, wrapper } = setup();
    const { result } = renderHook(
      () => useUpdatePolicyAutomations({ policy, onError }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ policyUpdate: { automation_type: null } });
    });

    await waitFor(() => expect(onError).toHaveBeenCalled());
    expect(client.getQueryState(["config"])?.isInvalidated).toBe(false);
  });
});
