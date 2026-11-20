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
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return { wrapper };
};

describe("useUpdatePolicyAutomations", () => {
  afterEach(() => jest.restoreAllMocks());

  it("saves the automation in one update to the policy", async () => {
    const updateSpy = jest
      .spyOn(policiesAPI, "update")
      .mockResolvedValue({} as never);
    const onSuccess = jest.fn();
    const { wrapper } = setup();
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
  });

  it("reports a failed save", async () => {
    jest.spyOn(policiesAPI, "update").mockRejectedValue(new Error("nope"));
    const onError = jest.fn();
    const { wrapper } = setup();
    const { result } = renderHook(
      () => useUpdatePolicyAutomations({ policy, onError }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ policyUpdate: { automation_type: null } });
    });

    await waitFor(() => expect(onError).toHaveBeenCalled());
  });
});
