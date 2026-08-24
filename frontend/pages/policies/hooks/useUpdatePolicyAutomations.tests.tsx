import React from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "react-query";

import { AppContext, IAppContext, initialState } from "context/app";
import createMockConfig from "__mocks__/configMock";
import createMockPolicy from "__mocks__/policyMock";
import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import useUpdatePolicyAutomations from "./useUpdatePolicyAutomations";

const buildWrapper = (appOverrides: Partial<IAppContext> = {}) => {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const value: IAppContext = { ...initialState, ...appOverrides };
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>
      <AppContext.Provider value={value}>{children}</AppContext.Provider>
    </QueryClientProvider>
  );
  return { Wrapper, client };
};

// Empty webhook baseline: the mutation reads existing `policy_ids` off this
// object to compute the next list. `ITeamConfig` is bigger than we need, so we
// cast a minimal shape to `ITeamConfig` for the team-scoped tests.
const emptyGlobalConfig: IConfig = createMockConfig({
  webhook_settings: {
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "",
      policy_ids: [],
      host_batch_size: 0,
    },
    activities_webhook: {
      enable_activities_webhook: false,
      destination_url: "",
    },
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 0,
      days_count: 0,
    },
    interval: "0s",
    vulnerabilities_webhook: {
      enable_vulnerabilities_webhook: false,
      destination_url: "",
      host_batch_size: 0,
    },
  },
});

const emptyTeamConfig = ({
  webhook_settings: {
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "",
      policy_ids: [],
      host_batch_size: 0,
    },
  },
} as unknown) as ITeamConfig;

describe("useUpdatePolicyAutomations", () => {
  beforeEach(() => {
    jest.restoreAllMocks();
  });

  it("primes the ['config'] cache from the write response when a global policy toggles a webhook", async () => {
    // The read-after-write regression (#42546): pages used to invalidate
    // ["config"] here, triggering a refetch that returned stale data under
    // replica lag. The fix consumes the PATCH response directly.
    const policy = createMockPolicy({ id: 42, team_id: null });
    const updated = createMockConfig({
      ...emptyGlobalConfig,
      webhook_settings: {
        ...emptyGlobalConfig.webhook_settings,
        failing_policies_webhook: {
          ...emptyGlobalConfig.webhook_settings.failing_policies_webhook,
          policy_ids: [42],
        },
      },
    });
    const setConfig = jest.fn();
    const configUpdate = jest
      .spyOn(configAPI, "update")
      .mockResolvedValue(updated);

    const { Wrapper, client } = buildWrapper({ setConfig });
    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: undefined,
          isGlobalPolicy: true,
          automationsConfig: emptyGlobalConfig,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current.mutateAsync({
        webhookOrTicketUpdate: { enabled: true },
      });
    });

    expect(configUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        webhook_settings: expect.objectContaining({
          failing_policies_webhook: expect.objectContaining({
            policy_ids: [42],
          }),
        }),
      })
    );
    expect(setConfig).toHaveBeenCalledWith(updated);
    expect(client.getQueryData(["config"])).toBe(updated);
  });

  it("primes the ['teams', id] cache from the write response when a team policy toggles a webhook", async () => {
    const policy = createMockPolicy({ id: 99, team_id: 3 });
    const updatedTeam = ({
      team: { ...emptyTeamConfig, id: 3 },
    } as unknown) as ILoadTeamResponse;
    const teamsUpdate = jest
      .spyOn(teamsAPI, "update")
      .mockResolvedValue(updatedTeam);

    const { Wrapper, client } = buildWrapper();
    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: 3,
          isGlobalPolicy: false,
          automationsConfig: emptyTeamConfig,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current.mutateAsync({
        webhookOrTicketUpdate: { enabled: true },
      });
    });

    expect(teamsUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        webhook_settings: expect.objectContaining({
          failing_policies_webhook: expect.objectContaining({
            policy_ids: [99],
          }),
        }),
      }),
      3
    );
    expect(client.getQueryData(["teams", 3])).toBe(updatedTeam);
  });

  it("removes the policy id when the webhook is toggled off", async () => {
    const policy = createMockPolicy({ id: 7, team_id: null });
    const configWithMembership = createMockConfig({
      webhook_settings: {
        ...emptyGlobalConfig.webhook_settings,
        failing_policies_webhook: {
          ...emptyGlobalConfig.webhook_settings.failing_policies_webhook,
          policy_ids: [7, 11],
        },
      },
    });
    const configUpdate = jest
      .spyOn(configAPI, "update")
      .mockResolvedValue(configWithMembership);

    const { Wrapper } = buildWrapper({ setConfig: jest.fn() });
    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: undefined,
          isGlobalPolicy: true,
          automationsConfig: configWithMembership,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current.mutateAsync({
        webhookOrTicketUpdate: { enabled: false },
      });
    });

    expect(configUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        webhook_settings: expect.objectContaining({
          failing_policies_webhook: expect.objectContaining({
            policy_ids: [11],
          }),
        }),
      })
    );
  });

  it("sends per-policy fields via teamPoliciesAPI.update and does not touch the config cache", async () => {
    const policy = createMockPolicy({ id: 5, team_id: null });
    const policyUpdate = jest
      .spyOn(teamPoliciesAPI, "update")
      .mockResolvedValue({ policy });
    const setConfig = jest.fn();
    const configUpdate = jest.spyOn(configAPI, "update");

    const { Wrapper, client } = buildWrapper({ setConfig });
    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: undefined,
          isGlobalPolicy: true,
          automationsConfig: emptyGlobalConfig,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current.mutateAsync({
        policyUpdate: { calendar_events_enabled: true },
      });
    });

    expect(policyUpdate).toHaveBeenCalledWith(
      5,
      expect.objectContaining({ calendar_events_enabled: true })
    );
    expect(configUpdate).not.toHaveBeenCalled();
    expect(setConfig).not.toHaveBeenCalled();
    expect(client.getQueryData(["config"])).toBeUndefined();
  });

  it("invalidates ['config'] on error for a global policy so a stale primed cache does not stick", async () => {
    const policy = createMockPolicy({ id: 1, team_id: null });
    jest.spyOn(configAPI, "update").mockRejectedValue(new Error("boom"));
    const onError = jest.fn();

    const { Wrapper, client } = buildWrapper({ setConfig: jest.fn() });
    const invalidateSpy = jest.spyOn(client, "invalidateQueries");

    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: undefined,
          isGlobalPolicy: true,
          automationsConfig: emptyGlobalConfig,
          onError,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current
        .mutateAsync({ webhookOrTicketUpdate: { enabled: true } })
        .catch(() => undefined);
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith(["config"]);
    });
    expect(onError).toHaveBeenCalled();
  });

  it("invalidates ['teams', id] on error for a team-scoped policy", async () => {
    const policy = createMockPolicy({ id: 2, team_id: 4 });
    jest.spyOn(teamsAPI, "update").mockRejectedValue(new Error("boom"));

    const { Wrapper, client } = buildWrapper();
    const invalidateSpy = jest.spyOn(client, "invalidateQueries");

    const { result } = renderHook(
      () =>
        useUpdatePolicyAutomations({
          policy,
          teamIdForApi: 4,
          isGlobalPolicy: false,
          automationsConfig: emptyTeamConfig,
        }),
      { wrapper: Wrapper }
    );

    await act(async () => {
      await result.current
        .mutateAsync({ webhookOrTicketUpdate: { enabled: true } })
        .catch(() => undefined);
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith(["teams", 4]);
    });
  });

  it("throws at hook-init time when a team-scoped policy is missing teamIdForApi", () => {
    const policy = createMockPolicy({ id: 1, team_id: 1 });
    const { Wrapper } = buildWrapper();

    // Silence React's error-boundary console noise for this expected throw.
    const errSpy = jest
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    expect(() =>
      renderHook(
        () =>
          useUpdatePolicyAutomations({
            policy,
            teamIdForApi: undefined,
            isGlobalPolicy: false,
            automationsConfig: emptyTeamConfig,
          }),
        { wrapper: Wrapper }
      )
    ).toThrow(/Missing fleet id/);

    errSpy.mockRestore();
  });
});
